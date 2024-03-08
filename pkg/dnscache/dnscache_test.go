package dnscache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/nettest"
)

func TestNew(t *testing.T) {
	t.Parallel()

	got := New(nil, 3, 5*time.Second)
	require.NotNil(t, got)

	require.NotNil(t, got.resolver)
	require.NotNil(t, got.mux)

	require.Equal(t, 3, got.size)
	require.Equal(t, 5*time.Second, got.ttl)

	require.NotNil(t, got.hostmap)
	require.Empty(t, got.hostmap)

	got = New(nil, 0, 1*time.Second)
	require.Equal(t, 1, got.size)
}

func Test_evict_expired(t *testing.T) {
	t.Parallel()

	r := New(nil, 3, 1*time.Minute)

	r.hostmap = map[string]*entry{
		"example.com": {
			expireAt: time.Now().UTC().Add(-2 * time.Second).Unix(),
		},
		"example.org": {
			expireAt: time.Now().UTC().Add(11 * time.Second).Unix(),
		},
		"example.net": {
			expireAt: time.Now().UTC().Add(13 * time.Second).Unix(),
		},
	}

	require.Len(t, r.hostmap, 3)

	r.evict()

	require.Len(t, r.hostmap, 2)
	require.Contains(t, r.hostmap, "example.org")
	require.Contains(t, r.hostmap, "example.net")
}

func Test_evict_oldest(t *testing.T) {
	t.Parallel()

	c := New(nil, 3, 1*time.Second)

	c.hostmap = map[string]*entry{
		"example.com": {
			expireAt: time.Now().UTC().Add(11 * time.Second).Unix(),
		},
		"example.org": {
			expireAt: time.Now().UTC().Add(7 * time.Second).Unix(),
		},
		"example.net": {
			expireAt: time.Now().UTC().Add(13 * time.Second).Unix(),
		},
	}

	c.evict()

	require.Len(t, c.hostmap, 2)
	require.Contains(t, c.hostmap, "example.com")
	require.Contains(t, c.hostmap, "example.net")
}

/*
NOTE:
The IP blocks 192.0.2.0/24 (TEST-NET-1), 198.51.100.0/24 (TEST-NET-2),
and 203.0.113.0/24 (TEST-NET-3) are provided for use in documentation.
*/

func Test_set(t *testing.T) {
	t.Parallel()

	c := New(nil, 2, 10*time.Second)

	c.set("example.com", []string{"192.0.2.1"}, nil, nil)
	time.Sleep(1 * time.Second)
	c.set("example.org", []string{"192.0.2.2", "198.51.100.2"}, nil, nil)

	require.Len(t, c.hostmap, 2)
	require.Contains(t, c.hostmap, "example.com")
	require.Contains(t, c.hostmap, "example.org")

	c.set("example.net", []string{"192.0.2.3", "198.51.100.3", "203.0.113.3"}, nil, nil)

	require.Len(t, c.hostmap, 2)
	require.Contains(t, c.hostmap, "example.org")
	require.Contains(t, c.hostmap, "example.net")

	c.set("example.net", []string{"198.51.100.4"}, nil, nil)

	require.Len(t, c.hostmap, 2)
	require.Contains(t, c.hostmap, "example.org")
	require.Contains(t, c.hostmap, "example.net")
	require.Equal(t, []string{"198.51.100.4"}, c.hostmap["example.net"].addrs)
}

type mockResolver struct {
	lookupHost func(ctx context.Context, host string) ([]string, error)
}

func (m *mockResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	return m.lookupHost(ctx, host)
}

func Test_LookupHost(t *testing.T) {
	t.Parallel()

	var i int

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			i++
			ip := fmt.Sprintf("192.0.2.%d", i)
			return []string{ip}, nil
		},
	}

	c := New(resolver, 1, 1*time.Second)

	// cache miss
	addrs, err := c.LookupHost(context.TODO(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.1"}, addrs)

	// cache hit
	addrs, err = c.LookupHost(context.TODO(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.1"}, addrs)

	time.Sleep(1 * time.Second)

	// cache expired
	addrs, err = c.LookupHost(context.TODO(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.2"}, addrs)

	// cache miss with eviction
	addrs, err = c.LookupHost(context.TODO(), "example.net")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.3"}, addrs)

	// deleted entry on duplicate lookup
	wait := make(chan struct{})

	c.mux.Lock()
	c.set("example.org", nil, nil, wait)
	c.mux.Unlock()

	go func() {
		time.Sleep(5 * time.Millisecond)
		c.Remove("example.org")
		close(wait)
	}()

	addrs, err = c.LookupHost(context.TODO(), "example.org")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.4"}, addrs)

	// context expired on duplicate lookup
	wait = make(chan struct{})

	c.mux.Lock()
	c.set("example.org", nil, nil, wait)
	c.mux.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	c.set("example.org", nil, nil, wait)

	addrs, err = c.LookupHost(ctx, "example.org")
	require.Error(t, err)
	require.Nil(t, addrs)
}

func Test_LookupHost_concurrent_slow(t *testing.T) {
	t.Parallel()

	const nlookup = 10

	type retval struct {
		err   error
		addrs []string
	}

	var i int

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			time.Sleep(300 * time.Millisecond) // simulate slow lookup
			i++
			ip := fmt.Sprintf("192.0.2.%d", i)
			return []string{ip}, nil
		},
	}

	c := New(resolver, 2, 0)
	ret := make(chan retval, nlookup)
	wg := &sync.WaitGroup{}

	for j := 0; j < nlookup; j++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			addrs, err := c.LookupHost(context.TODO(), "example.org")
			ret <- retval{err, addrs}
		}()
	}

	go func() {
		wg.Wait()
		close(ret)
	}()

	for v := range ret {
		require.NoError(t, v.err)
		require.NotNil(t, v.addrs)
		require.Len(t, v.addrs, 1)
		require.Equal(t, []string{"192.0.2.1"}, v.addrs)
	}
}

func Test_LookupHost_concurrent_fast(t *testing.T) {
	t.Parallel()

	const nlookup = 1234

	type retval struct {
		err   error
		addrs []string
	}

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			return []string{"192.0.2.13"}, nil
		},
	}

	// With ttl = 0 the items expires immediately causing stress on the concurrent lookups.
	// This covers the case when the cache entry was updated during the wait.
	// This should not happen in real world scenarios, but it's good to have it covered.

	c := New(resolver, 2, 0)
	ret := make(chan retval, nlookup)
	wg := &sync.WaitGroup{}

	for j := 0; j < nlookup; j++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			addrs, err := c.LookupHost(context.TODO(), "example.org")
			ret <- retval{err, addrs}
		}()
	}

	go func() {
		wg.Wait()
		close(ret)
	}()

	for v := range ret {
		require.NoError(t, v.err)
		require.NotNil(t, v.addrs)
		require.Len(t, v.addrs, 1)
		require.Equal(t, []string{"192.0.2.13"}, v.addrs)
	}
}

func Test_LookupHost_error(t *testing.T) {
	t.Parallel()

	const nlookup = 10

	type retval struct {
		err   error
		addrs []string
	}

	var i int

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			time.Sleep(300 * time.Millisecond) // simulate slow lookup
			i++
			return nil, fmt.Errorf("mock error: %d", i)
		},
	}

	c := New(resolver, 2, 10*time.Second)

	addrs, err := c.LookupHost(context.TODO(), "example.com")
	require.Error(t, err)
	require.Nil(t, addrs)

	// test concurrent lookups

	ret := make(chan retval, nlookup)
	wg := &sync.WaitGroup{}

	for j := 0; j < nlookup; j++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			addrs, err := c.LookupHost(context.TODO(), "example.net")
			ret <- retval{err, addrs}
		}()
	}

	go func() {
		wg.Wait()
		close(ret)
	}()

	for v := range ret {
		require.Error(t, v.err)
		require.Equal(t, "mock error: 2", v.err.Error())
		require.Nil(t, v.addrs)
	}
}

func Test_LookupHost_error_concurrent_fast(t *testing.T) {
	t.Parallel()

	const nlookup = 100

	type retval struct {
		err   error
		addrs []string
	}

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			return nil, errors.New("mock error")
		},
	}

	c := New(resolver, 2, 0)

	ret := make(chan retval, nlookup)
	wg := &sync.WaitGroup{}

	for j := 0; j < nlookup; j++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			addrs, err := c.LookupHost(context.TODO(), "example.net")
			ret <- retval{err, addrs}
		}()
	}

	go func() {
		wg.Wait()
		close(ret)
	}()

	for v := range ret {
		require.Error(t, v.err)
		require.Equal(t, "mock error", v.err.Error())
		require.Nil(t, v.addrs)
	}
}

func Test_DialContext_lookup_errors(t *testing.T) {
	t.Parallel()

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			return nil, errors.New("mock error")
		},
	}

	c := New(resolver, 1, 1*time.Second)

	// SplitHostPort error
	conn, err := c.DialContext(context.TODO(), "tcp", "~~~")
	require.Error(t, err)
	require.Nil(t, conn)

	// LookupHost error
	conn, err = c.DialContext(context.TODO(), "tcp", "example.com:80")
	require.Error(t, err)
	require.Nil(t, conn)
}

func Test_DialContext_ip_error(t *testing.T) {
	t.Parallel()

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			return []string{"1"}, nil
		},
	}

	c := New(resolver, 1, 1*time.Second)

	conn, err := c.DialContext(context.TODO(), "tcp", "example.com:80")
	require.Error(t, err)
	require.Nil(t, conn)
}

func Test_DialContext(t *testing.T) {
	t.Parallel()

	network := "tcp"

	listener, err := nettest.NewLocalListener(network)
	require.NoError(t, err)
	require.NotNil(t, listener)

	defer func() {
		err := listener.Close()
		require.NoError(t, err)
	}()

	address := listener.Addr().String()
	addrparts := strings.Split(address, ":")
	ip := addrparts[0]

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			return []string{ip}, nil
		},
	}

	r := New(resolver, 1, 1*time.Second)

	conn, err := r.DialContext(context.TODO(), network, address)
	require.NoError(t, err)
	require.NotNil(t, conn)
}

func Test_Reset(t *testing.T) {
	t.Parallel()

	c := New(nil, 1, 1*time.Second)

	c.hostmap = map[string]*entry{
		"example.com": {
			expireAt: time.Now().UTC().Unix(),
		},
	}

	c.Reset()

	require.Empty(t, c.hostmap)
}

func Test_Remove(t *testing.T) {
	t.Parallel()

	c := New(nil, 3, 1*time.Second)

	c.hostmap = map[string]*entry{
		"example.com": {
			expireAt: time.Now().UTC().Unix(),
		},
		"example.net": {
			expireAt: time.Now().UTC().Unix(),
		},
		"example.org": {
			expireAt: time.Now().UTC().Unix(),
		},
	}

	c.Remove("example.net")

	require.Len(t, c.hostmap, 2)
	require.Contains(t, c.hostmap, "example.com")
	require.Contains(t, c.hostmap, "example.org")
}
