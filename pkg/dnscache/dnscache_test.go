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
	require.NotNil(t, got.cache)
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

	for range nlookup {
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

	for range nlookup {
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

	for range nlookup {
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

	for range nlookup {
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
