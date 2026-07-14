package dnscache

import (
	"context"
	"errors"
	"fmt"
	"net"
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
	addrs, err := c.LookupHost(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.1"}, addrs)

	// cache hit
	addrs, err = c.LookupHost(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.1"}, addrs)

	time.Sleep(1 * time.Second)

	// cache expired
	addrs, err = c.LookupHost(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.2"}, addrs)

	// cache miss with eviction
	addrs, err = c.LookupHost(t.Context(), "example.net")
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
		wg.Go(func() {
			addrs, err := c.LookupHost(t.Context(), "example.org")
			ret <- retval{err, addrs}
		})
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
		wg.Go(func() {
			addrs, err := c.LookupHost(t.Context(), "example.org")
			ret <- retval{err, addrs}
		})
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

	addrs, err := c.LookupHost(t.Context(), "example.com")
	require.Error(t, err)
	require.Nil(t, addrs)

	// test concurrent lookups

	ret := make(chan retval, nlookup)
	wg := &sync.WaitGroup{}

	for range nlookup {
		wg.Go(func() {
			addrs, err := c.LookupHost(t.Context(), "example.net")
			ret <- retval{err, addrs}
		})
	}

	go func() {
		wg.Wait()
		close(ret)
	}()

	for v := range ret {
		require.Error(t, v.err)
		require.Equal(t, "unable to resolve example.net: DNS lookup failed: mock error: 2", v.err.Error())
		require.Nil(t, v.addrs)
	}
}

func Test_LookupHost_error_not_cached(t *testing.T) {
	t.Parallel()

	var i int

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			i++

			if i == 1 {
				return nil, errors.New("transient mock error")
			}

			return []string{"192.0.2.1"}, nil
		},
	}

	c := New(resolver, 2, 1*time.Minute)

	// A transient DNS failure must not be negatively cached.
	addrs, err := c.LookupHost(t.Context(), "example.com")
	require.Error(t, err)
	require.Nil(t, addrs)

	// The next call must re-query the resolver and succeed within the TTL.
	addrs, err = c.LookupHost(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.1"}, addrs)
	require.Equal(t, 2, i)
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
		wg.Go(func() {
			addrs, err := c.LookupHost(t.Context(), "example.net")
			ret <- retval{err, addrs}
		})
	}

	go func() {
		wg.Wait()
		close(ret)
	}()

	for v := range ret {
		require.Error(t, v.err)
		require.Equal(t, "unable to resolve example.net: DNS lookup failed: mock error", v.err.Error())
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
	conn, err := c.DialContext(t.Context(), "tcp", "~~~")
	require.Error(t, err)
	require.Nil(t, conn)

	// LookupHost error
	conn, err = c.DialContext(t.Context(), "tcp", "example.com:80")
	require.Error(t, err)
	require.Nil(t, conn)
}

func Test_DialContext_no_addresses(t *testing.T) {
	t.Parallel()

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			return []string{}, nil
		},
	}

	c := New(resolver, 1, 1*time.Second)

	conn, err := c.DialContext(t.Context(), "tcp", "example.com:80")
	require.Error(t, err)
	require.Nil(t, conn)
	require.ErrorIs(t, err, ErrNoAddresses)
	require.Equal(t, "dnscache: no addresses resolved for example.com", err.Error())
}

func Test_DialContext_ip_error(t *testing.T) {
	t.Parallel()

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			return []string{"1"}, nil
		},
	}

	c := New(resolver, 1, 1*time.Second)

	conn, err := c.DialContext(t.Context(), "tcp", "example.com:80")
	require.Error(t, err)
	require.Nil(t, conn)
	require.ErrorIs(t, err, ErrInvalidIP)
}

func Test_DialContext_failover(t *testing.T) {
	t.Parallel()

	network := "tcp"

	// Bind to IPv4 loopback explicitly so the first (failing) candidate and the
	// live listener share an address family and the same port.
	var lc net.ListenConfig

	listener, err := lc.Listen(t.Context(), network, "127.0.0.1:0")
	require.NoError(t, err)
	require.NotNil(t, listener)

	defer func() {
		err := listener.Close()
		require.NoError(t, err)
	}()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			// Nothing listens on IPv6 loopback at this port, so "::1" fails
			// fast on every platform (refused, or unsupported without IPv6);
			// the dial loop must fall through to the live listener on
			// 127.0.0.1. The mixed families also exercise the interleave path
			// end-to-end.
			return []string{"::1", "127.0.0.1"}, nil
		},
	}

	c := New(resolver, 1, 1*time.Second)

	// A non-IP host name is required so the resolver (not the IP-literal
	// bypass) supplies the candidate list.
	conn, err := c.DialContext(t.Context(), network, net.JoinHostPort("failover.test", port))
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.NoError(t, conn.Close())
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

	conn, err := r.DialContext(t.Context(), network, address)
	require.NoError(t, err)
	require.NotNil(t, conn)
}

func Test_Len(t *testing.T) {
	t.Parallel()

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			return []string{"192.0.2.1"}, nil
		},
	}

	c := New(resolver, 3, 1*time.Second)

	// cache miss
	addrs, err := c.LookupHost(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.1"}, addrs)

	require.Equal(t, 1, c.Len())

	// cache miss
	addrs, err = c.LookupHost(t.Context(), "example.net")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.1"}, addrs)

	require.Equal(t, 2, c.Len())
}

func Test_Reset(t *testing.T) {
	t.Parallel()

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			return []string{"192.0.2.1"}, nil
		},
	}

	c := New(resolver, 3, 1*time.Second)

	// cache miss
	addrs, err := c.LookupHost(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.1"}, addrs)

	// cache miss
	addrs, err = c.LookupHost(t.Context(), "example.net")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.1"}, addrs)

	require.Equal(t, 2, c.Len())

	c.Reset()

	require.Empty(t, c.Len())
}

func Test_Remove(t *testing.T) {
	t.Parallel()

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			return []string{"192.0.2.1"}, nil
		},
	}

	c := New(resolver, 3, 1*time.Minute)

	// cache miss
	addrs, err := c.LookupHost(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.1"}, addrs)

	// cache miss
	addrs, err = c.LookupHost(t.Context(), "example.net")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.1"}, addrs)

	require.Equal(t, 2, c.Len())

	c.Remove("example.net")

	require.Equal(t, 1, c.Len())
}

// Test_LookupHost_returns_copy verifies that mutating the returned slice does
// not corrupt the cached entry shared with other callers.
func Test_LookupHost_returns_copy(t *testing.T) {
	t.Parallel()

	var i int

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			i++

			return []string{"192.0.2.1", "192.0.2.2"}, nil
		},
	}

	c := New(resolver, 2, 1*time.Minute)

	addrs, err := c.LookupHost(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.1", "192.0.2.2"}, addrs)

	// Mutating the returned slice must not corrupt the cached entry.
	addrs[0] = "0.0.0.0"

	addrs, err = c.LookupHost(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.1", "192.0.2.2"}, addrs)
	require.Equal(t, 1, i, "the second call must be served from cache")
}

func Test_PurgeExpired(t *testing.T) {
	t.Parallel()

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			return []string{"192.0.2.1"}, nil
		},
	}

	c := New(resolver, 3, 100*time.Millisecond)

	_, err := c.LookupHost(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, 1, c.Len())

	require.Equal(t, 0, c.PurgeExpired(), "a fresh entry must not be purged")

	time.Sleep(150 * time.Millisecond) // let the entry expire

	require.Equal(t, 1, c.PurgeExpired())
	require.Equal(t, 0, c.Len())
}

func Test_LookupHost_normalizes_host(t *testing.T) {
	t.Parallel()

	var calls int

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, host string) ([]string, error) {
			calls++
			// The resolver must receive the normalized host name.
			require.Equal(t, "example.com", host)

			return []string{"192.0.2.1"}, nil
		},
	}

	c := New(resolver, 3, 1*time.Minute)

	// Every case and trailing-dot variant must collapse onto one cache entry.
	for _, h := range []string{"example.com", "EXAMPLE.com", "Example.Com.", "example.com."} {
		addrs, err := c.LookupHost(t.Context(), h)
		require.NoError(t, err)
		require.Equal(t, []string{"192.0.2.1"}, addrs)
	}

	require.Equal(t, 1, calls, "all variants must share a single cached lookup")
	require.Equal(t, 1, c.Len())

	// Remove must apply the same normalization.
	c.Remove("EXAMPLE.COM.")
	require.Equal(t, 0, c.Len())
}

func Test_normalizeHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		host string
		want string
	}{
		{name: "already normalized", host: "example.com", want: "example.com"},
		{name: "uppercase folded", host: "EXAMPLE.com", want: "example.com"},
		{name: "mixed case", host: "Example.COM", want: "example.com"},
		{name: "trailing dot stripped", host: "example.com.", want: "example.com"},
		{name: "case and trailing dot", host: "EXAMPLE.COM.", want: "example.com"},
		{name: "root kept", host: ".", want: "."},
		{name: "empty kept", host: "", want: ""},
		{name: "punycode untouched", host: "xn--80ak6aa92e.com", want: "xn--80ak6aa92e.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, normalizeHost(tt.host))
		})
	}
}

// rawsOf extracts the raw address strings from a candidate slice for assertions.
func rawsOf(cands []dialCandidate) []string {
	out := make([]string, len(cands))
	for i, c := range cands {
		out[i] = c.raw
	}

	return out
}

func Test_newCandidates_dedupe(t *testing.T) {
	t.Parallel()

	// Duplicate addresses are collapsed, preserving first-seen order.
	got := rawsOf(newCandidates([]string{"192.0.2.1", "192.0.2.1", "192.0.2.2", "192.0.2.1"}))
	require.Equal(t, []string{"192.0.2.1", "192.0.2.2"}, got)

	// Equivalent spellings collapse onto the first occurrence: case-variant
	// IPv6 and an IPv4-mapped IPv6 form of an IPv4 address.
	got = rawsOf(newCandidates([]string{"2001:db8::1", "2001:DB8::1", "192.0.2.1", "::ffff:192.0.2.1"}))
	require.Equal(t, []string{"2001:db8::1", "192.0.2.1"}, got)

	// Invalid entries de-duplicate on the raw string and never match a valid
	// address.
	got = rawsOf(newCandidates([]string{"notanip", "notanip", "other", "192.0.2.1"}))
	require.Equal(t, []string{"notanip", "other", "192.0.2.1"}, got)
}

func Test_rotateCandidates(t *testing.T) {
	t.Parallel()

	// Slices shorter than two elements are returned unchanged.
	single := newCandidates([]string{"192.0.2.1"})
	require.Equal(t, []string{"192.0.2.1"}, rawsOf(rotateCandidates(single, 7)))

	cands := newCandidates([]string{"192.0.2.1", "192.0.2.2", "192.0.2.3"})

	// Offset zero keeps the resolver order.
	require.Equal(t, []string{"192.0.2.1", "192.0.2.2", "192.0.2.3"}, rawsOf(rotateCandidates(cands, 0)))

	// A non-zero offset rotates the first-tried address.
	require.Equal(t, []string{"192.0.2.2", "192.0.2.3", "192.0.2.1"}, rawsOf(rotateCandidates(cands, 1)))

	// The offset wraps around the slice length.
	require.Equal(t, []string{"192.0.2.1", "192.0.2.2", "192.0.2.3"}, rawsOf(rotateCandidates(cands, 3)))
}

func Test_orderCandidates_family_order(t *testing.T) {
	t.Parallel()

	c := New(nil, 1, 1*time.Minute)

	// IPv6-led input keeps IPv6 first: the resolver's RFC 6724 family
	// preference must be preserved, not overridden.
	got := rawsOf(c.orderCandidates("tcp", []string{"2001:db8::1", "2001:db8::2", "192.0.2.1"}))
	require.Equal(t, []string{"2001:db8::1", "192.0.2.1", "2001:db8::2"}, got)

	// IPv4-led input keeps IPv4 first.
	got = rawsOf(c.orderCandidates("tcp", []string{"192.0.2.1", "192.0.2.2", "2001:db8::1"}))
	require.Equal(t, []string{"192.0.2.1", "2001:db8::1", "192.0.2.2"}, got)

	// A single-family list (a non-IP entry stays with the IPv4 group) is
	// returned unchanged.
	got = rawsOf(c.orderCandidates("tcp", []string{"192.0.2.1", "notanip", "192.0.2.2"}))
	require.Equal(t, []string{"192.0.2.1", "notanip", "192.0.2.2"}, got)
}

func Test_filterByNetwork(t *testing.T) {
	t.Parallel()

	mixed := []string{"2001:db8::1", "192.0.2.1", "notanip", "192.0.2.2"}

	// Family-neutral networks and an empty network keep every candidate.
	for _, network := range []string{"tcp", "udp", ""} {
		got := rawsOf(filterByNetwork(newCandidates(mixed), network))
		require.Equal(t, mixed, got, "network %q must keep all candidates", network)
	}

	// IPv4-restricted networks keep IPv4 and invalid entries.
	for _, network := range []string{"tcp4", "udp4"} {
		got := rawsOf(filterByNetwork(newCandidates(mixed), network))
		require.Equal(t, []string{"192.0.2.1", "notanip", "192.0.2.2"}, got, "network %q", network)
	}

	// IPv6-restricted networks keep IPv6 and invalid entries.
	for _, network := range []string{"tcp6", "udp6"} {
		got := rawsOf(filterByNetwork(newCandidates(mixed), network))
		require.Equal(t, []string{"2001:db8::1", "notanip"}, got, "network %q", network)
	}
}

func Test_LookupHost_ip_literal(t *testing.T) {
	t.Parallel()

	called := false

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			called = true

			return nil, errors.New("resolver must not be called for an IP literal")
		},
	}

	c := New(resolver, 1, 1*time.Minute)

	// An IP-literal host bypasses both the resolver and the cache.
	addrs, err := c.LookupHost(t.Context(), "192.0.2.1")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.1"}, addrs)
	require.False(t, called)
	require.Equal(t, 0, c.Len(), "IP literals must not be cached")
}

func Test_DialContext_context_canceled(t *testing.T) {
	t.Parallel()

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			return []string{"192.0.2.1", "192.0.2.2"}, nil
		},
	}

	c := New(resolver, 1, 1*time.Minute)

	// Warm the cache so the lookup is a context-independent hit and the dial
	// loop is reached even with a canceled context.
	_, err := c.LookupHost(t.Context(), "example.com")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	conn, err := c.DialContext(ctx, "tcp", "example.com:80")
	require.Error(t, err)
	require.Nil(t, conn)
	require.ErrorIs(t, err, context.Canceled)
}

func Test_DialContext_rotation(t *testing.T) {
	t.Parallel()

	c := New(nil, 4, 1*time.Minute, WithAddressRotation())

	ips := []string{"192.0.2.1", "192.0.2.2", "192.0.2.3", "192.0.2.4"}

	// With rotation enabled, successive calls advance the first-tried address.
	first := rawsOf(c.orderCandidates("tcp", ips))[0]
	second := rawsOf(c.orderCandidates("tcp", ips))[0]
	require.NotEqual(t, first, second)
}

func Test_orderCandidates_rotation_keeps_lead_family(t *testing.T) {
	t.Parallel()

	c := New(nil, 4, 1*time.Minute, WithAddressRotation())

	ips := []string{"2001:db8::1", "2001:db8::2", "192.0.2.1"}
	leads := make(map[string]bool)

	// Rotation happens within each family: the IPv6 lead must never flip to
	// IPv4, while the first-tried IPv6 address must still vary across calls.
	for range 6 {
		got := rawsOf(c.orderCandidates("tcp", ips))
		require.Contains(t, []string{"2001:db8::1", "2001:db8::2"}, got[0])

		leads[got[0]] = true
	}

	require.Len(t, leads, 2, "rotation must vary the first-tried address")
}

func Test_DialContext_family_restricted(t *testing.T) {
	t.Parallel()

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			return []string{"2001:db8::1", "2001:db8::2"}, nil
		},
	}

	c := New(resolver, 1, 1*time.Minute)

	// An IPv6-only host on an IPv4-restricted network has no usable address:
	// the error must say so instead of burning doomed dial attempts.
	conn, err := c.DialContext(t.Context(), "tcp4", "example.com:80")
	require.Error(t, err)
	require.Nil(t, conn)
	require.ErrorIs(t, err, ErrNoAddresses)
	require.Equal(t, "dnscache: no addresses resolved for example.com on network tcp4", err.Error())
}

func Test_DialContext_family_restricted_dial(t *testing.T) {
	t.Parallel()

	network := "tcp4"

	var lc net.ListenConfig

	listener, err := lc.Listen(t.Context(), network, "127.0.0.1:0")
	require.NoError(t, err)

	defer func() {
		err := listener.Close()
		require.NoError(t, err)
	}()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			// The IPv6 candidate is unusable on tcp4 and must be filtered out,
			// not dialed: the IPv4 listener address must be tried first.
			return []string{"2001:db8::1", "127.0.0.1"}, nil
		},
	}

	c := New(resolver, 1, 1*time.Minute)

	conn, err := c.DialContext(t.Context(), network, net.JoinHostPort("v4only.test", port))
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.NoError(t, conn.Close())
}

func Test_DialContext_with_dial_timeout(t *testing.T) {
	t.Parallel()

	network := "tcp"

	var lc net.ListenConfig

	listener, err := lc.Listen(t.Context(), network, "127.0.0.1:0")
	require.NoError(t, err)

	defer func() {
		err := listener.Close()
		require.NoError(t, err)
	}()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			return []string{"127.0.0.1"}, nil
		},
	}

	// A generous per-attempt timeout must not interfere with a fast local dial.
	c := New(resolver, 1, 1*time.Minute, WithDialTimeout(5*time.Second))

	conn, err := c.DialContext(t.Context(), network, net.JoinHostPort("timeout.test", port))
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.NoError(t, conn.Close())
}

// fakeConn is a no-op net.Conn used to stub out dialing in tests; its methods
// are never called.
type fakeConn struct {
	net.Conn
}

func Test_DialContext_dials_canonical(t *testing.T) {
	t.Parallel()

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			// An IPv4-mapped IPv6 spelling of an IPv4 address.
			return []string{"::ffff:127.0.0.1"}, nil
		},
	}

	c := New(resolver, 1, 1*time.Minute)

	var dialed string

	c.dialCtx = func(_ context.Context, _, address string) (net.Conn, error) {
		dialed = address

		return fakeConn{}, nil
	}

	conn, err := c.DialContext(t.Context(), "tcp", "example.com:80")
	require.NoError(t, err)
	require.NotNil(t, conn)

	// The mapped literal must be dialed in its canonical, unmapped IPv4 form.
	require.Equal(t, "127.0.0.1:80", dialed)
}

func Test_DialContext_dial_timeout_fires(t *testing.T) {
	t.Parallel()

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			return []string{"192.0.2.1", "192.0.2.2"}, nil
		},
	}

	c := New(resolver, 1, 1*time.Minute, WithDialTimeout(10*time.Millisecond))

	var attempts int

	// The first attempt blocks until its per-attempt context is canceled; the
	// timeout must fire and advance the loop to the second address, which
	// succeeds. dialCtx is called sequentially within one DialContext, so a
	// plain counter is race-free.
	c.dialCtx = func(ctx context.Context, _, _ string) (net.Conn, error) {
		attempts++
		if attempts == 1 {
			<-ctx.Done()

			return nil, ctx.Err()
		}

		return fakeConn{}, nil
	}

	conn, err := c.DialContext(t.Context(), "tcp", "example.com:80")
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 2, attempts, "the per-attempt timeout must advance to the next address")
}

func Test_DialContext_concurrent(t *testing.T) {
	t.Parallel()

	network := "tcp"

	var lc net.ListenConfig

	listener, err := lc.Listen(t.Context(), network, "127.0.0.1:0")
	require.NoError(t, err)

	defer func() {
		err := listener.Close()
		require.NoError(t, err)
	}()

	go func() {
		for {
			conn, aerr := listener.Accept()
			if aerr != nil {
				return
			}

			_ = conn.Close()
		}
	}()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			return []string{"::1", "127.0.0.1"}, nil
		},
	}

	// Rotation exercises the shared atomic counter under concurrency (run with -race).
	c := New(resolver, 1, 1*time.Minute, WithAddressRotation())

	wg := &sync.WaitGroup{}

	for range 50 {
		wg.Go(func() {
			conn, derr := c.DialContext(t.Context(), network, net.JoinHostPort("concurrent.test", port))
			if derr == nil {
				_ = conn.Close()
			}
		})
	}

	wg.Wait()
}

// Test_New_honors_the_configured_size pins that the size passed to New is the
// capacity the underlying cache is actually built with: nothing else in this
// package's tests would notice an off-by-one in that wiring.
func Test_New_honors_the_configured_size(t *testing.T) {
	t.Parallel()

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, host string) ([]string, error) {
			return []string{"192.0.2." + host[:1]}, nil
		},
	}

	const size = 3

	c := New(resolver, size, time.Hour)

	for i := range 4 * size {
		_, err := c.LookupHost(t.Context(), fmt.Sprintf("%dhost.example.com", i%10))
		require.NoError(t, err)

		require.LessOrEqual(t, c.Len(), size, "the cache must never hold more than the configured size")
	}
}
