// Tests for stale-if-error.

package sfcache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_Lookup_stale_if_error_serves_and_recovers drives the full
// stale-if-error lifecycle with both options combined: a failed refresh
// serves the last known good value with a nil error, repeated failures keep
// serving it, and the first successful refresh replaces it. The value-driven
// ttlFn gives the first value a short TTL (to expire it quickly) and the
// recovered value a long one (so the final cache-hit assertion has a wide
// timing window), and makes the stale window derive from the per-entry TTL.
func Test_Lookup_stale_if_error_serves_and_recovers(t *testing.T) {
	t.Parallel()

	var i int

	lookupFn := func(_ context.Context, _ string) (any, error) {
		i++

		if i == 2 || i == 3 {
			return nil, errors.New("mock refresh failure")
		}

		return i, nil
	}

	ttlFn := func(_ string, v any) time.Duration {
		if v.(int) == 1 { //nolint:forcetypeassert
			return 200 * time.Millisecond // the first value expires quickly
		}

		return 1 * time.Minute // recovered values stay fresh
	}

	c := New(lookupFn, Config{Size: 4, TTL: 0, MaxStale: 1 * time.Minute}, WithTTLFunc(ttlFn))

	// Initial successful lookup.
	v, err := c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, 1, v)

	time.Sleep(250 * time.Millisecond) // let the entry expire

	// First failed refresh: the stale value is served with a nil error.
	v, err = c.Lookup(t.Context(), "example.com")
	require.NoError(t, err, "a failed refresh within maxStale must serve the stale value")
	require.Equal(t, 1, v)
	require.Equal(t, 2, i, "the refresh must have been attempted")

	// Second failed refresh: the revived entry is expired, so the lookup is
	// attempted again and the stale value is served again.
	v, err = c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, 1, v)
	require.Equal(t, 3, i)

	// Successful refresh: recovery replaces the stale value.
	v, err = c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, 4, v)

	// The recovered value is cached again: no further lookup.
	v, err = c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, 4, v)
	require.Equal(t, 4, i)
}

// Test_Lookup_stale_if_error_bound verifies that the stale value is not
// served past its original expiration plus maxStale.
func Test_Lookup_stale_if_error_bound(t *testing.T) {
	t.Parallel()

	var i int

	lookupFn := func(_ context.Context, _ string) (any, error) {
		i++

		if i > 1 {
			return nil, errors.New("mock refresh failure")
		}

		return i, nil
	}

	c := New(lookupFn, Config{Size: 4, TTL: 200 * time.Millisecond, MaxStale: 200 * time.Millisecond})

	v, err := c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, 1, v)

	// Sleep past expiration (200ms) plus maxStale (200ms).
	time.Sleep(650 * time.Millisecond)

	_, err = c.Lookup(t.Context(), "example.com")
	require.Error(t, err, "the stale value must not be served past its maxStale bound")
	require.Equal(t, 2, i)
}

// Test_Lookup_stale_if_error_shared_with_waiters verifies that coalesced
// waiters of a failed refresh receive the same revived stale value as the
// caller that performed the lookup.
func Test_Lookup_stale_if_error_shared_with_waiters(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	var calls atomic.Int32

	lookupFn := func(_ context.Context, _ string) (any, error) {
		if calls.Add(1) == 1 {
			return "good", nil
		}

		<-release // hold the failing refresh open

		return nil, errors.New("mock refresh failure")
	}

	c := New(lookupFn, Config{Size: 4, TTL: 100 * time.Millisecond, MaxStale: 1 * time.Minute})

	v, err := c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, "good", v)

	time.Sleep(150 * time.Millisecond) // let the entry expire

	prodDone := make(chan struct{})

	go func() {
		defer close(prodDone)

		v, err := c.Lookup(context.Background(), "example.com")
		assert.NoError(t, err) // assert (not require) must be used off the test goroutine
		assert.Equal(t, "good", v, "the producer must receive the revived stale value")
	}()

	// Wait until the refresh flight has registered its flight.
	require.Eventually(t, func() bool {
		return inFlight(c, "example.com")
	}, time.Second, time.Millisecond)

	waiterDone := make(chan struct{})

	go func() {
		defer close(waiterDone)

		v, err := c.Lookup(context.Background(), "example.com")
		assert.NoError(t, err) // assert (not require) must be used off the test goroutine
		assert.Equal(t, "good", v, "waiters must receive the revived stale value")
	}()

	waitForParkedLookupWaiter(t, "stale_if_error_shared_with_waiters")
	close(release)

	<-prodDone
	<-waiterDone

	require.Equal(t, int32(2), calls.Load(), "the failed refresh must be a single flight")
}

// Test_Lookup_stale_if_error_requires_prior_success verifies that failures
// are returned as errors when no previous good value exists, including after
// an error residue entry.
func Test_Lookup_stale_if_error_requires_prior_success(t *testing.T) {
	t.Parallel()

	var i int

	lookupFn := func(_ context.Context, _ string) (any, error) {
		i++

		return nil, fmt.Errorf("mock error: %d", i)
	}

	c := New(lookupFn, Config{Size: 4, TTL: 1 * time.Minute, MaxStale: 1 * time.Minute})

	// No previous value: the error is returned.
	_, err := c.Lookup(t.Context(), "example.com")
	require.Error(t, err)
	require.Equal(t, "mock error: 1", err.Error())

	// The error residue must not be served stale either.
	_, err = c.Lookup(t.Context(), "example.com")
	require.Error(t, err)
	require.Equal(t, "mock error: 2", err.Error())
	require.Equal(t, 2, i)
}

// Test_Lookup_stale_if_error_context_failure_serves_stale verifies that the
// stale window takes precedence over the context-induced retry: a refresh
// that fails because the producing caller's context ended still serves the
// last known good value to the producer and to coalesced waiters, so an
// upstream that hangs (making every refresh die by caller timeout) is served
// stale instead of never firing stale-if-error at all.
func Test_Lookup_stale_if_error_context_failure_serves_stale(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	var calls atomic.Int32

	lookupFn := func(ctx context.Context, _ string) (any, error) {
		if calls.Add(1) == 1 {
			return "good", nil
		}

		<-release // hold the refresh open until canceled

		return nil, fmt.Errorf("lookup: %w", ctx.Err())
	}

	c := New(lookupFn, Config{Size: 4, TTL: 100 * time.Millisecond, MaxStale: 1 * time.Minute})

	v, err := c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, "good", v)

	time.Sleep(150 * time.Millisecond) // let the entry expire

	pctx, pcancel := context.WithCancel(context.Background())
	prodDone := make(chan struct{})

	go func() {
		defer close(prodDone)

		v, err := c.Lookup(pctx, "example.com")
		assert.NoError(t, err) // assert (not require) must be used off the test goroutine
		assert.Equal(t, "good", v, "the producer must receive the stale value despite its dead context")
	}()

	// Wait until the refresh flight has registered its flight.
	require.Eventually(t, func() bool {
		return inFlight(c, "example.com")
	}, time.Second, time.Millisecond)

	var (
		waiterVal any
		waiterErr error
	)

	waiterDone := make(chan struct{})

	go func() {
		defer close(waiterDone)

		waiterVal, waiterErr = c.Lookup(context.Background(), "example.com")
	}()

	waitForParkedLookupWaiter(t, "stale_if_error_context_failure_serves_stale")

	pcancel()
	close(release)

	<-prodDone
	<-waiterDone

	require.NoError(t, waiterErr)
	require.Equal(t, "good", waiterVal, "waiters must be served stale instead of retrying a hanging upstream")
	require.Equal(t, int32(2), calls.Load(), "no retry must happen when stale is served")
}

// Test_Lookup_stale_if_error_context_failure_outside_window_retries verifies
// that a context-induced failure outside the stale window keeps its retry
// semantics: a live waiter retries the lookup with its own context.
func Test_Lookup_stale_if_error_context_failure_outside_window_retries(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	var calls atomic.Int32

	lookupFn := func(ctx context.Context, _ string) (any, error) {
		switch calls.Add(1) {
		case 1:
			return "good", nil
		case 2:
			<-release // hold the refresh open until canceled

			return nil, fmt.Errorf("lookup: %w", ctx.Err())
		default:
			return "fresh", nil
		}
	}

	// The stale window (expiry + 50ms) is already past when the refresh runs.
	c := New(lookupFn, Config{Size: 4, TTL: 100 * time.Millisecond, MaxStale: 50 * time.Millisecond})

	v, err := c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, "good", v)

	time.Sleep(300 * time.Millisecond) // sleep past expiration (100ms) plus maxStale (50ms)

	pctx, pcancel := context.WithCancel(context.Background())
	prodDone := make(chan struct{})

	go func() {
		defer close(prodDone)

		_, err := c.Lookup(pctx, "example.com")
		assert.ErrorIs(t, err, context.Canceled) // assert (not require) must be used off the test goroutine
	}()

	// Wait until the refresh flight has registered its flight.
	require.Eventually(t, func() bool {
		return inFlight(c, "example.com")
	}, time.Second, time.Millisecond)

	var (
		waiterVal any
		waiterErr error
	)

	waiterDone := make(chan struct{})

	go func() {
		defer close(waiterDone)

		waiterVal, waiterErr = c.Lookup(context.Background(), "example.com")
	}()

	waitForParkedLookupWaiter(t, "stale_if_error_context_failure_outside_window_retries")

	pcancel()
	close(release)

	<-prodDone
	<-waiterDone

	require.NoError(t, waiterErr)
	require.Equal(t, "fresh", waiterVal, "outside the stale window a live waiter must retry")
	require.Equal(t, int32(3), calls.Load())
}

// Test_Lookup_stale_if_error_original_deadline verifies that repeated failed
// refreshes preserve the ORIGINAL stale deadline (expiration + maxStale):
// stale is served late-but-inside the window and refused past it, so
// continuous failure cannot extend staleness forever.
func Test_Lookup_stale_if_error_original_deadline(t *testing.T) {
	t.Parallel()

	var i int

	lookupFn := func(_ context.Context, _ string) (any, error) {
		i++

		if i == 1 {
			return "good", nil
		}

		return nil, errors.New("mock refresh failure")
	}

	// The stale window closes at expiration (t0+200ms) + maxStale (600ms).
	c := New(lookupFn, Config{Size: 4, TTL: 200 * time.Millisecond, MaxStale: 600 * time.Millisecond})

	v, err := c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, "good", v)

	time.Sleep(250 * time.Millisecond) // ~t0+250: expired, inside the window

	v, err = c.Lookup(t.Context(), "example.com")
	require.NoError(t, err, "a failure inside the stale window must serve stale")
	require.Equal(t, "good", v)

	time.Sleep(250 * time.Millisecond) // ~t0+500: still inside the ORIGINAL window

	v, err = c.Lookup(t.Context(), "example.com")
	require.NoError(t, err, "a repeated failure inside the original window must serve stale")
	require.Equal(t, "good", v)

	time.Sleep(400 * time.Millisecond) // ~t0+900: past the original window (t0+800)

	_, err = c.Lookup(t.Context(), "example.com")
	require.Error(t, err, "repeated failures must not extend the original stale deadline")
	require.Equal(t, 4, i)
}

// Test_Lookup_stale_if_error_with_zero_ttl verifies that with ttl <= 0 the
// previous value is still retained for stale-if-error: value caching is
// disabled, but a failed refresh serves the last known good value.
func Test_Lookup_stale_if_error_with_zero_ttl(t *testing.T) {
	t.Parallel()

	var i int

	lookupFn := func(_ context.Context, _ string) (any, error) {
		i++

		if i == 1 {
			return "good", nil
		}

		return nil, errors.New("mock refresh failure")
	}

	c := New(lookupFn, Config{Size: 2, TTL: 0, MaxStale: 1 * time.Minute})

	v, err := c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, "good", v)

	// Value caching is disabled (ttl 0), so this triggers a refresh, which
	// fails: the previous value must be served stale.
	v, err = c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, "good", v)
	require.Equal(t, 2, i)
}

// Test_Lookup_stale_if_error_lost_to_eviction pins the documented best-effort
// nature of stale protection: a revived stale entry is expired and therefore
// evicted first under capacity pressure, after which failures surface as
// errors again.
func Test_Lookup_stale_if_error_lost_to_eviction(t *testing.T) {
	t.Parallel()

	var i int

	lookupFn := func(_ context.Context, key string) (any, error) {
		i++

		if key == "other.example.com" {
			return "other", nil
		}

		if i == 1 {
			return "good", nil
		}

		return nil, errors.New("mock refresh failure")
	}

	c := New(lookupFn, Config{Size: 1, TTL: 100 * time.Millisecond, MaxStale: 1 * time.Minute})

	v, err := c.Lookup(t.Context(), "stale.example.com")
	require.NoError(t, err)
	require.Equal(t, "good", v)

	time.Sleep(150 * time.Millisecond) // let the entry expire

	v, err = c.Lookup(t.Context(), "stale.example.com")
	require.NoError(t, err)
	require.Equal(t, "good", v, "the failed refresh must serve stale")

	// Inserting another key at capacity evicts the revived (expired) entry.
	v, err = c.Lookup(t.Context(), "other.example.com")
	require.NoError(t, err)
	require.Equal(t, "other", v)

	_, err = c.Lookup(t.Context(), "stale.example.com")
	require.Error(t, err, "after eviction the stale value must be gone")
}

// Test_Lookup_stale_if_error_lost_to_purge pins that PurgeExpired removes
// revived stale entries, forfeiting stale protection.
func Test_Lookup_stale_if_error_lost_to_purge(t *testing.T) {
	t.Parallel()

	var i int

	lookupFn := func(_ context.Context, _ string) (any, error) {
		i++

		if i == 1 {
			return "good", nil
		}

		return nil, errors.New("mock refresh failure")
	}

	c := New(lookupFn, Config{Size: 2, TTL: 100 * time.Millisecond, MaxStale: 1 * time.Minute})

	_, err := c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)

	time.Sleep(150 * time.Millisecond) // let the entry expire

	v, err := c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, "good", v, "the failed refresh must serve stale")

	require.Equal(t, 1, c.PurgeExpired(), "the revived stale entry must be purged")

	_, err = c.Lookup(t.Context(), "example.com")
	require.Error(t, err, "after PurgeExpired the stale value must be gone")
}

// Test_Lookup_stale_if_error_lost_to_remove pins that Remove discards the
// stale value: explicit invalidation wins over stale protection.
func Test_Lookup_stale_if_error_lost_to_remove(t *testing.T) {
	t.Parallel()

	var i int

	lookupFn := func(_ context.Context, _ string) (any, error) {
		i++

		if i == 1 {
			return "good", nil
		}

		return nil, errors.New("mock refresh failure")
	}

	c := New(lookupFn, Config{Size: 2, TTL: 100 * time.Millisecond, MaxStale: 1 * time.Minute})

	_, err := c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)

	time.Sleep(150 * time.Millisecond) // let the entry expire

	v, err := c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, "good", v, "the failed refresh must serve stale")

	c.Remove("example.com")

	_, err = c.Lookup(t.Context(), "example.com")
	require.Error(t, err, "after Remove the stale value must be gone")
}

// Test_Lookup_stale_on_failure_cold_key pins the difference between the two
// stale windows: for a key idle for longer than ttl + maxStale, the
// expiration-anchored window (MaxStale) is closed and the error is returned,
// while the failure-anchored one (MaxStaleOnFailure) still serves the last
// known good value.
func Test_Lookup_stale_on_failure_cold_key(t *testing.T) {
	t.Parallel()

	newCache := func(cfg Config) *Cache[string, any] {
		var i int

		return New(func(_ context.Context, _ string) (any, error) {
			i++

			if i == 1 {
				return "good", nil
			}

			return nil, errors.New("mock refresh failure")
		}, cfg)
	}

	// idle > ttl + maxStale for both caches
	const (
		ttl      = 20 * time.Millisecond
		maxStale = 20 * time.Millisecond
		idle     = 120 * time.Millisecond
	)

	expiryAnchored := newCache(Config{Size: 4, TTL: ttl, MaxStale: maxStale})
	failureAnchored := newCache(Config{Size: 4, TTL: ttl, MaxStale: maxStale, MaxStaleOnFailure: 1 * time.Minute})

	for _, c := range []*Cache[string, any]{expiryAnchored, failureAnchored} {
		v, err := c.Lookup(t.Context(), "example.com")
		require.NoError(t, err)
		require.Equal(t, "good", v)
	}

	time.Sleep(idle)

	_, err := expiryAnchored.Lookup(t.Context(), "example.com")
	require.Error(t, err, "MaxStale is anchored to the expiration: a cold key has no stale protection")

	v, err := failureAnchored.Lookup(t.Context(), "example.com")
	require.NoError(t, err, "MaxStaleOnFailure is anchored to the failure: a cold key is still protected")
	require.Equal(t, "good", v)
}

// Test_Lookup_stale_on_failure_window_is_anchored_once verifies that the
// failure-anchored window is fixed by the first failed refresh and is not
// pushed back by the failures that follow: a permanently failing upstream
// cannot make a value immortal.
func Test_Lookup_stale_on_failure_window_is_anchored_once(t *testing.T) {
	t.Parallel()

	var i int

	lookupFn := func(_ context.Context, _ string) (any, error) {
		i++

		if i == 1 {
			return "good", nil
		}

		return nil, errors.New("mock refresh failure")
	}

	// The window is deliberately much longer than the sleeps that must land
	// inside it (150ms of slack) and much shorter than the one that must land
	// outside it (150ms again), so that a scheduling stall cannot flip either
	// assertion.
	c := New(lookupFn, Config{Size: 4, TTL: 20 * time.Millisecond, MaxStaleOnFailure: 400 * time.Millisecond})

	_, err := c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond) // let the entry expire

	// First failure: anchors the window at now + 400ms.
	v, err := c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, "good", v)

	// Further failures keep serving the same value, without re-anchoring.
	time.Sleep(250 * time.Millisecond)

	v, err = c.Lookup(t.Context(), "example.com")
	require.NoError(t, err, "the value must still be served inside the anchored window")
	require.Equal(t, "good", v)

	// Past the deadline anchored by the first failure, the error surfaces:
	// had the second failure re-anchored the window, this would still be stale.
	time.Sleep(300 * time.Millisecond)

	_, err = c.Lookup(t.Context(), "example.com")
	require.Error(t, err, "the window must not be extended by later failures")
}

// Test_Lookup_stale_windows_combined verifies that when both windows are
// configured the value is served until the LATER of the two deadlines. Each case
// sets one window to a minute and the other to a millisecond, then keeps serving
// well past the shorter one: only the longer window can account for that, so
// neither "the expiration anchor always wins" nor "the failure anchor always
// wins" can pass both.
func Test_Lookup_stale_windows_combined(t *testing.T) {
	t.Parallel()

	newCache := func(cfg Config) *Cache[string, any] {
		var i int

		return New(func(_ context.Context, _ string) (any, error) {
			i++

			if i == 1 {
				return "good", nil
			}

			return nil, errors.New("mock refresh failure")
		}, cfg)
	}

	tests := []struct {
		name string
		cfg  Config
	}{
		{
			// expiry + 1min outlives failure + 1ms.
			name: "the expiration-anchored window is the later one",
			cfg: Config{
				Size:              4,
				TTL:               20 * time.Millisecond,
				MaxStale:          1 * time.Minute,
				MaxStaleOnFailure: 1 * time.Millisecond,
			},
		},
		{
			// failure + 1min outlives expiry + 1ms, which is closed by the time
			// the refresh fails.
			name: "the failure-anchored window is the later one",
			cfg: Config{
				Size:              4,
				TTL:               20 * time.Millisecond,
				MaxStale:          1 * time.Millisecond,
				MaxStaleOnFailure: 1 * time.Minute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newCache(tt.cfg)

			_, err := c.Lookup(t.Context(), "example.com")
			require.NoError(t, err)

			time.Sleep(50 * time.Millisecond) // let the entry expire

			// The first failed refresh anchors the window and serves stale.
			v, err := c.Lookup(t.Context(), "example.com")
			require.NoError(t, err)
			require.Equal(t, "good", v)

			// Well past the SHORTER of the two deadlines: only the later one
			// can still be serving this value.
			time.Sleep(100 * time.Millisecond)

			v, err = c.Lookup(t.Context(), "example.com")
			require.NoError(t, err, "the later of the two deadlines must win")
			require.Equal(t, "good", v)
		})
	}
}

// Test_Lookup_failing_key_does_not_destroy_a_stale_value pins that a key which
// is merely attempted, and fails, cannot take a value that is actively being
// served stale through an upstream outage.
func Test_Lookup_failing_key_does_not_destroy_a_stale_value(t *testing.T) {
	t.Parallel()

	var down atomic.Bool

	lookupFn := func(_ context.Context, key string) (any, error) {
		switch {
		case key == "bad.example.com":
			return nil, errors.New("mock error")
		case down.Load() && (key == "stale.example.com"):
			return nil, errors.New("mock refresh failure")
		}

		return key, nil
	}

	c := New(lookupFn, Config{Size: 2, TTL: 50 * time.Millisecond, MaxStaleOnFailure: time.Minute})

	_, err := c.Lookup(t.Context(), "stale.example.com")
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond) // let the entry expire

	down.Store(true)

	v, err := c.Lookup(t.Context(), "stale.example.com")
	require.NoError(t, err)
	require.Equal(t, "stale.example.com", v, "the failed refresh must revive the stale value")

	// Bring the cache to capacity, then attempt a key whose lookup fails.
	_, err = c.Lookup(t.Context(), "good.example.com")
	require.NoError(t, err)

	_, err = c.Lookup(t.Context(), "bad.example.com")
	require.Error(t, err)

	v, err = c.Lookup(t.Context(), "stale.example.com")
	require.NoError(t, err, "a failed lookup must not destroy a value that is being served stale")
	require.Equal(t, "stale.example.com", v)

	requireConsistentAccounting(t, c)
}

// Test_Lookup_error_residue_is_never_served_stale pins that a key that has never
// succeeded is never served stale. With [Config.MaxStaleOnFailure] the window
// opens at the failure, so without the guard a key whose every lookup fails
// would return its zero value with a nil error.
func Test_Lookup_error_residue_is_never_served_stale(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	lookupFn := func(_ context.Context, _ string) (any, error) {
		calls.Add(1)

		return nil, errors.New("mock error")
	}

	c := New(lookupFn, Config{Size: 2, TTL: time.Minute, MaxStaleOnFailure: time.Minute})

	for range 3 {
		v, err := c.Lookup(t.Context(), "example.com")
		require.Error(t, err, "a key that never succeeded must never be served stale")
		require.Nil(t, v)
	}

	require.Equal(t, int32(3), calls.Load(), "errors are never cached")
}

// Test_Lookup_negative_ttl_keeps_the_stale_window pins that a negative TTL
// behaves exactly like a zero one: the value expires as it is stored, never
// before it existed, so it cannot eat into the [Config.MaxStale] window that is
// anchored to its expiration.
func Test_Lookup_negative_ttl_keeps_the_stale_window(t *testing.T) {
	t.Parallel()

	for _, ttl := range []time.Duration{0, -100 * time.Millisecond, -time.Hour} {
		var calls int

		lookupFn := func(_ context.Context, _ string) (any, error) {
			calls++

			if calls == 1 {
				return "good", nil
			}

			return nil, errors.New("mock refresh failure")
		}

		c := New(lookupFn, Config{Size: 2, TTL: ttl, MaxStale: time.Minute})

		_, err := c.Lookup(t.Context(), "example.com")
		require.NoError(t, err)

		v, err := c.Lookup(t.Context(), "example.com")
		require.NoErrorf(t, err, "a TTL of %s must not close the stale window", ttl)
		require.Equal(t, "good", v)
	}
}

// Test_Lookup_closed_max_stale_window_survives_a_failing_key pins the case order
// inside worthlessAt, in the only state that can observe it: a cache with BOTH
// windows, holding a value whose MaxStale window has closed while its
// MaxStaleOnFailure window can still revive it. Such a value is NOT worthless,
// and a lookup that produces no cacheable value must not destroy it.
func Test_Lookup_closed_max_stale_window_survives_a_failing_key(t *testing.T) {
	t.Parallel()

	var down atomic.Bool

	lookupFn := func(_ context.Context, key string) (string, error) {
		if down.Load() || strings.HasPrefix(key, "bad") {
			return "", errors.New("upstream down")
		}

		return "good-" + key, nil
	}

	// A MaxStale window that shuts almost immediately, and a MaxStaleOnFailure
	// window that stays wide open.
	c := New(lookupFn, Config{
		Size:              2,
		TTL:               5 * time.Millisecond,
		MaxStale:          10 * time.Millisecond,
		MaxStaleOnFailure: time.Hour,
	})

	for _, key := range []string{"a", "b"} {
		v, err := c.Lookup(t.Context(), key)
		require.NoError(t, err)
		require.Equal(t, "good-"+key, v)
	}

	// Let both the TTL and the MaxStale window lapse. Only MaxStaleOnFailure can
	// protect these values now — and it still can, because neither has yet met a
	// failure.
	time.Sleep(30 * time.Millisecond)

	down.Store(true)

	// A storm of always-failing lookups of OTHER keys, each of which produces no
	// cacheable value and so may reclaim only worthless entries.
	for i := range 30 {
		_, err := c.Lookup(t.Context(), fmt.Sprintf("bad%d", i))
		require.Error(t, err)
	}

	// Both values must have survived, and must still be served stale.
	for _, key := range []string{"a", "b"} {
		v, err := c.Lookup(t.Context(), key)
		require.NoErrorf(t, err, "%q was destroyed by a lookup that produced no value, "+
			"while MaxStaleOnFailure was still protecting it", key)
		require.Equal(t, "good-"+key, v)
	}
}

// Test_staleFrom_retains_nothing_without_a_stale_window pins the guard that stops
// a cache with no stale-if-error from copying V into the staleState and holding
// it for the whole duration of every flight.
func Test_staleFrom_retains_nothing_without_a_stale_window(t *testing.T) {
	t.Parallel()

	live := &entry[any]{val: "value", expireAt: time.Now().Add(-time.Second)}

	c := New(nopLookupFn, Config{Size: 2, TTL: time.Minute}) // no stale window

	stale := c.staleFrom(live)
	require.False(t, stale.ok, "a cache with no stale-if-error must capture no stale value")
	require.Nil(t, stale.val, "and must not retain the value for the duration of the flight")

	// The control: with a window configured, the same entry IS captured.
	sc := New(nopLookupFn, Config{Size: 2, TTL: time.Minute, MaxStaleOnFailure: time.Hour})

	stale = sc.staleFrom(live)
	require.True(t, stale.ok)
	require.Equal(t, "value", stale.val)
}
