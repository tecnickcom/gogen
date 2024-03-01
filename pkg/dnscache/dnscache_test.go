package dnscache

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

	require.NotNil(t, got.cache)
	require.Empty(t, got.cache)

	got = New(nil, 0, 1*time.Second)
	require.Equal(t, 1, got.size)
}

func Test_evict_expired(t *testing.T) {
	t.Parallel()

	r := New(nil, 3, 1*time.Minute)

	r.cache = map[string]*dnsItem{
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

	require.Len(t, r.cache, 3)

	r.evict()

	require.Len(t, r.cache, 2)
	require.Contains(t, r.cache, "example.org")
	require.Contains(t, r.cache, "example.net")
}

func Test_evict_oldest(t *testing.T) {
	t.Parallel()

	r := New(nil, 3, 1*time.Second)

	r.cache = map[string]*dnsItem{
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

	r.evict()

	require.Len(t, r.cache, 2)
	require.Contains(t, r.cache, "example.com")
	require.Contains(t, r.cache, "example.net")
}

/*
NOTE:
The IP blocks 192.0.2.0/24 (TEST-NET-1), 198.51.100.0/24 (TEST-NET-2),
and 203.0.113.0/24 (TEST-NET-3) are provided for use in documentation.
*/

func Test_set(t *testing.T) {
	t.Parallel()

	r := New(nil, 2, 10*time.Second)

	r.set("example.com", []string{"192.0.2.1"}, false)
	time.Sleep(1 * time.Second)
	r.set("example.org", []string{"192.0.2.2", "198.51.100.2"}, false)

	require.Len(t, r.cache, 2)
	require.Contains(t, r.cache, "example.com")
	require.Contains(t, r.cache, "example.org")

	r.set("example.net", []string{"192.0.2.3", "198.51.100.3", "203.0.113.3"}, false)

	require.Len(t, r.cache, 2)
	require.Contains(t, r.cache, "example.org")
	require.Contains(t, r.cache, "example.net")

	r.set("example.net", []string{"198.51.100.4"}, true)

	require.Len(t, r.cache, 2)
	require.Contains(t, r.cache, "example.org")
	require.Contains(t, r.cache, "example.net")
	require.Equal(t, []string{"198.51.100.4"}, r.cache["example.net"].addrs)
}

type mockResolver struct {
	lookupHost func(ctx context.Context, host string) ([]string, error)
}

func (m *mockResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	return m.lookupHost(ctx, host)
}

func Test_LookupHost_error(t *testing.T) {
	t.Parallel()

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			return nil, errors.New("mock error")
		},
	}

	r := New(resolver, 1, 1*time.Second)

	addrs, err := r.LookupHost(context.TODO(), "example.com")
	require.Error(t, err)
	require.Nil(t, addrs)
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

	r := New(resolver, 1, 1*time.Second)

	// cache miss
	addrs, err := r.LookupHost(context.TODO(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.1"}, addrs)

	// cache hit
	addrs, err = r.LookupHost(context.TODO(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.1"}, addrs)

	time.Sleep(1 * time.Second)

	// cache expired
	addrs, err = r.LookupHost(context.TODO(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.2"}, addrs)

	// cache miss with eviction
	addrs, err = r.LookupHost(context.TODO(), "example.net")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.3"}, addrs)
}

func Test_DialContext_lookup_errors(t *testing.T) {
	t.Parallel()

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			return nil, errors.New("mock error")
		},
	}

	r := New(resolver, 1, 1*time.Second)

	// SplitHostPort error
	conn, err := r.DialContext(context.TODO(), "tcp", "~~~")
	require.Error(t, err)
	require.Nil(t, conn)

	// LookupHost error
	conn, err = r.DialContext(context.TODO(), "tcp", "example.com:80")
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

	r := New(resolver, 1, 1*time.Second)

	conn, err := r.DialContext(context.TODO(), "tcp", "example.com:80")
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

	r := New(nil, 1, 1*time.Second)

	r.cache = map[string]*dnsItem{
		"example.com": {
			expireAt: time.Now().UTC().Unix(),
		},
	}

	r.Reset()

	require.Empty(t, r.cache)
}

func Test_RemoveEntry(t *testing.T) {
	t.Parallel()

	r := New(nil, 3, 1*time.Second)

	r.cache = map[string]*dnsItem{
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

	r.RemoveEntry("example.net")

	require.Len(t, r.cache, 2)
	require.Contains(t, r.cache, "example.com")
	require.Contains(t, r.cache, "example.org")
}
