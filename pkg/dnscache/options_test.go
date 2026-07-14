package dnscache

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_WithDialer(t *testing.T) {
	t.Parallel()

	cfg := &config{dialer: &net.Dialer{KeepAlive: defaultKeepAlive}}
	base := cfg.dialer

	// A nil dialer is ignored: the existing dialer is kept.
	WithDialer(nil)(cfg)
	require.Same(t, base, cfg.dialer)

	// A custom dialer replaces the default.
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	WithDialer(dialer)(cfg)
	require.Same(t, dialer, cfg.dialer)
}

func Test_WithDialTimeout(t *testing.T) {
	t.Parallel()

	cfg := &config{}
	require.Zero(t, cfg.dialTimeout)

	WithDialTimeout(3 * time.Second)(cfg)
	require.Equal(t, 3*time.Second, cfg.dialTimeout)
}

func Test_WithAddressRotation(t *testing.T) {
	t.Parallel()

	cfg := &config{}
	require.False(t, cfg.rotate)

	WithAddressRotation()(cfg)
	require.True(t, cfg.rotate)
}

func Test_WithStaleIfError(t *testing.T) {
	t.Parallel()

	var calls int

	resolver := &mockResolver{
		lookupHost: func(_ context.Context, _ string) ([]string, error) {
			calls++
			if calls == 1 {
				return []string{"192.0.2.1"}, nil
			}

			return nil, errors.New("resolver down")
		},
	}

	c := New(resolver, 2, 20*time.Millisecond, WithStaleIfError(1*time.Minute))

	addrs, err := c.LookupHost(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.1"}, addrs)

	time.Sleep(40 * time.Millisecond) // let the entry expire

	// The refresh fails, but the last known good value is served with no error.
	addrs, err = c.LookupHost(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.1"}, addrs)
	require.Equal(t, 2, calls)
}

// Test_WithStaleOnFailure verifies that the failure-anchored stale window
// protects a host that has been idle for far longer than ttl + maxStale, which
// is exactly the case WithStaleIfError does not cover.
//
// The two caches share the same timeline and differ only in the option, so the
// expiry-anchored one is the negative control: without it, this test would pass
// just as happily if the two options were wired to the same window.
func Test_WithStaleOnFailure(t *testing.T) {
	t.Parallel()

	newResolver := func() *mockResolver {
		var calls int

		return &mockResolver{
			lookupHost: func(_ context.Context, _ string) ([]string, error) {
				calls++
				if calls == 1 {
					return []string{"192.0.2.1"}, nil
				}

				return nil, errors.New("resolver down")
			},
		}
	}

	const (
		ttl      = 20 * time.Millisecond
		maxStale = 20 * time.Millisecond
	)

	expiryAnchored := New(newResolver(), 2, ttl, WithStaleIfError(maxStale))
	failureAnchored := New(newResolver(), 2, ttl, WithStaleOnFailure(1*time.Minute))

	for _, c := range []*Cache{expiryAnchored, failureAnchored} {
		addrs, err := c.LookupHost(t.Context(), "example.com")
		require.NoError(t, err)
		require.Equal(t, []string{"192.0.2.1"}, addrs)
	}

	// Idle well past ttl + maxStale: the RFC 5861 window is long closed.
	time.Sleep(120 * time.Millisecond)

	_, err := expiryAnchored.LookupHost(t.Context(), "example.com")
	require.Error(t, err, "WithStaleIfError is anchored to the expiry: an idle host is not protected")

	addrs, err := failureAnchored.LookupHost(t.Context(), "example.com")
	require.NoError(t, err, "WithStaleOnFailure is anchored to the failure: an idle host is still protected")
	require.Equal(t, []string{"192.0.2.1"}, addrs)
}
