// Tests for the lookup path: caching, coalescing, errors, TTLs, and context.

package sfcache

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Lookup(t *testing.T) {
	t.Parallel()

	var i int

	lookupFn := func(_ context.Context, _ string) (any, error) {
		i++

		ip := fmt.Sprintf("192.0.2.%d", i)

		return []string{ip}, nil
	}

	c := New(lookupFn, Config{Size: 1, TTL: 1 * time.Second})

	// cache miss
	val, err := c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.1"}, val)

	// cache hit
	val, err = c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.1"}, val)

	time.Sleep(1 * time.Second)

	// cache expired
	val, err = c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.2"}, val)

	// cache miss with eviction
	val, err = c.Lookup(t.Context(), "example.net")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.3"}, val)

	// deleted entry on duplicate lookup
	fl := seedFlight(c, "example.org")

	go func() {
		time.Sleep(5 * time.Millisecond)
		c.Remove("example.org") // this releases the waiter on its own
		fl.finish()             // idempotent: Remove already finished the flight
	}()

	val, err = c.Lookup(t.Context(), "example.org")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.4"}, val)

	// context expired on duplicate lookup: the flight is never finished, so the
	// waiter must be released by its own deadline.
	seedFlight(c, "example.org")

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()

	val, err = c.Lookup(ctx, "example.org")
	require.ErrorIs(t, err, ErrLookupAborted)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Nil(t, val)
}

func Test_Lookup_canceled_waiter_no_double_close(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	lookupFn := func(_ context.Context, _ string) (any, error) {
		<-release // hold the in-flight lookup open

		return []string{"192.0.2.1"}, nil
	}

	c := New(lookupFn, Config{Size: 4, TTL: time.Minute})

	prodDone := make(chan struct{})

	go func() {
		defer close(prodDone)

		v, err := c.Lookup(context.Background(), "example.org")
		assert.NoError(t, err) // assert (not require) must be used off the test goroutine
		assert.Equal(t, []string{"192.0.2.1"}, v)
	}()

	// Wait until the producer has registered the in-flight entry.
	require.Eventually(t, func() bool {
		return inFlight(c, "example.org")
	}, time.Second, time.Millisecond)

	// A waiter coalesces onto the producer, then its context is canceled.
	wctx, wcancel := context.WithCancel(context.Background())
	waiterDone := make(chan struct{})

	go func() {
		defer close(waiterDone)

		_, err := c.Lookup(wctx, "example.org")
		assert.Error(t, err) // assert (not require) must be used off the test goroutine
	}()

	waitForParkedLookupWaiter(t, "canceled_waiter_no_double_close")
	wcancel()
	<-waiterDone

	// The producer finishes and closes its own wait channel: must not panic
	// on a double close now that the canceled waiter no longer closes it.
	close(release)
	<-prodDone
}

func Test_Lookup_concurrent_slow(t *testing.T) {
	t.Parallel()

	const nlookup = 10

	type retval struct {
		err error
		val []string
	}

	release := make(chan struct{})

	var i atomic.Int32

	lookupFn := func(_ context.Context, _ string) (any, error) {
		<-release // hold the lookup open until every caller has joined

		ip := fmt.Sprintf("192.0.2.%d", i.Add(1))

		return []string{ip}, nil
	}

	c := New(lookupFn, Config{Size: 2, TTL: 0})
	ret := make(chan retval, nlookup)
	wg := &sync.WaitGroup{}

	for range nlookup {
		wg.Go(func() {
			val, err := c.Lookup(t.Context(), "example.org")

			v, ok := val.([]string)
			if !ok {
				ret <- retval{err, nil}
				return
			}

			ret <- retval{err, v}
		})
	}

	// All duplicate callers must be coalesced onto the single in-flight
	// lookup before it completes, regardless of scheduler load.
	waitForNParkedLookupWaiters(t, "concurrent_slow", nlookup-1)
	close(release)

	go func() {
		wg.Wait()
		close(ret)
	}()

	for v := range ret {
		require.NoError(t, v.err)
		require.NotNil(t, v.val)
		require.Len(t, v.val, 1)
		require.Equal(t, []string{"192.0.2.1"}, v.val)
	}
}

func Test_Lookup_concurrent_fast(t *testing.T) {
	t.Parallel()

	const nlookup = 1234

	type retval struct {
		err error
		val []string
	}

	lookupFn := func(_ context.Context, _ string) (any, error) {
		return []string{"192.0.2.13"}, nil
	}

	// With ttl = 0 the items expires immediately causing stress on the concurrent lookups.
	// This covers the case when the cache entry was updated during the wait.
	// This should not happen in real world scenarios, but it's good to have it covered.

	c := New(lookupFn, Config{Size: 2, TTL: 0})
	ret := make(chan retval, nlookup)
	wg := &sync.WaitGroup{}

	for range nlookup {
		wg.Go(func() {
			val, err := c.Lookup(t.Context(), "example.org")

			v, ok := val.([]string)
			if !ok {
				ret <- retval{err, nil}
				return
			}

			ret <- retval{err, v}
		})
	}

	go func() {
		wg.Wait()
		close(ret)
	}()

	for v := range ret {
		require.NoError(t, v.err)
		require.NotNil(t, v.val)
		require.Len(t, v.val, 1)
		require.Equal(t, []string{"192.0.2.13"}, v.val)
	}
}

func Test_Lookup_error(t *testing.T) {
	t.Parallel()

	const nlookup = 10

	type retval struct {
		err error
		val []string
	}

	release := make(chan struct{})

	var i atomic.Int32

	lookupFn := func(_ context.Context, key string) (any, error) {
		if key == "example.net" {
			<-release // hold the lookup open until every caller has joined
		}

		return nil, fmt.Errorf("mock error: %d", i.Add(1))
	}

	c := New(lookupFn, Config{Size: 2, TTL: 10 * time.Second})

	val, err := c.Lookup(t.Context(), "example.com")
	require.Error(t, err)
	require.Nil(t, val)

	// test concurrent lookups

	ret := make(chan retval, nlookup)
	wg := &sync.WaitGroup{}

	for range nlookup {
		wg.Go(func() {
			val, err := c.Lookup(t.Context(), "example.net")

			v, ok := val.([]string)
			if !ok {
				ret <- retval{err, nil}
				return
			}

			ret <- retval{err, v}
		})
	}

	// All duplicate callers must be coalesced onto the single in-flight
	// lookup before it fails, so that every caller shares its error.
	waitForNParkedLookupWaiters(t, "Test_Lookup_error.func", nlookup-1)
	close(release)

	go func() {
		wg.Wait()
		close(ret)
	}()

	for v := range ret {
		require.Error(t, v.err)
		require.Equal(t, "mock error: 2", v.err.Error())
		require.Nil(t, v.val)
	}
}

func Test_Lookup_error_with_typed_nil_value_not_cached(t *testing.T) {
	t.Parallel()

	var i int

	lookupFn := func(_ context.Context, _ string) (any, error) {
		i++

		// A typed-nil slice boxed into a non-nil `any` must still not be
		// cached when the lookup fails (no negative caching).
		return []string(nil), fmt.Errorf("mock error: %d", i)
	}

	c := New(lookupFn, Config{Size: 2, TTL: 10 * time.Second})

	_, err := c.Lookup(t.Context(), "example.com")
	require.Error(t, err)
	require.Equal(t, "mock error: 1", err.Error())

	_, err = c.Lookup(t.Context(), "example.com")
	require.Error(t, err)
	require.Equal(t, "mock error: 2", err.Error(), "errors must never be cached")
	require.Equal(t, 2, i)
}

func Test_Lookup_nil_success_cached(t *testing.T) {
	t.Parallel()

	var i int

	lookupFn := func(_ context.Context, _ string) (any, error) {
		i++

		return nil, nil //nolint:nilnil
	}

	c := New(lookupFn, Config{Size: 2, TTL: 10 * time.Second})

	val, err := c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Nil(t, val)

	val, err = c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Nil(t, val)

	require.Equal(t, 1, i, "nil-valued successes must be cached for the TTL")
}

func Test_Lookup_subsecond_ttl(t *testing.T) {
	t.Parallel()

	var i int

	lookupFn := func(_ context.Context, _ string) (any, error) {
		i++

		return fmt.Sprintf("192.0.2.%d", i), nil
	}

	c := New(lookupFn, Config{Size: 2, TTL: 500 * time.Millisecond})

	// cache miss
	val, err := c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, "192.0.2.1", val)

	// cache hit: a fresh entry with a sub-second TTL must not be already expired
	val, err = c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, "192.0.2.1", val)
	require.Equal(t, 1, i)

	time.Sleep(600 * time.Millisecond)

	// cache expired
	val, err = c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, "192.0.2.2", val)
	require.Equal(t, 2, i)
}

func Test_Lookup_error_concurrent_fast(t *testing.T) {
	t.Parallel()

	const nlookup = 100

	type retval struct {
		err error
		val []string
	}

	lookupFn := func(_ context.Context, _ string) (any, error) {
		return nil, errors.New("mock error")
	}

	c := New(lookupFn, Config{Size: 2, TTL: 0})

	ret := make(chan retval, nlookup)
	wg := &sync.WaitGroup{}

	for range nlookup {
		wg.Go(func() {
			val, err := c.Lookup(t.Context(), "example.net")

			v, ok := val.([]string)
			if !ok {
				ret <- retval{err, nil}
				return
			}

			ret <- retval{err, v}
		})
	}

	go func() {
		wg.Wait()
		close(ret)
	}()

	for v := range ret {
		require.Error(t, v.err)
		require.Equal(t, "mock error", v.err.Error())
		require.Nil(t, v.val)
	}
}

// Test_Lookup_producer_cancel_does_not_poison_waiters verifies that when the
// goroutine performing the lookup has its context canceled, the resulting
// error is not shared with coalesced waiters: one of them must retry the
// lookup with its own (live) context and succeed.
func Test_Lookup_producer_cancel_does_not_poison_waiters(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	var calls atomic.Int32

	lookupFn := func(ctx context.Context, _ string) (any, error) {
		if calls.Add(1) == 1 {
			<-release // hold the in-flight lookup open until canceled

			return nil, fmt.Errorf("lookup: %w", ctx.Err())
		}

		return "fresh", nil
	}

	c := New(lookupFn, Config{Size: 4, TTL: time.Minute})

	pctx, pcancel := context.WithCancel(context.Background())
	prodDone := make(chan struct{})

	go func() {
		defer close(prodDone)

		_, err := c.Lookup(pctx, "example.org")
		assert.ErrorIs(t, err, context.Canceled) // assert (not require) must be used off the test goroutine
	}()

	// Wait until the producer has registered the in-flight entry.
	require.Eventually(t, func() bool {
		return inFlight(c, "example.org")
	}, time.Second, time.Millisecond)

	// A waiter with a live context coalesces onto the producer.
	var (
		waiterVal any
		waiterErr error
	)

	waiterDone := make(chan struct{})

	go func() {
		defer close(waiterDone)

		waiterVal, waiterErr = c.Lookup(context.Background(), "example.org")
	}()

	waitForParkedLookupWaiter(t, "producer_cancel_does_not_poison_waiters")

	// Cancel the producer's context and let its lookup fail.
	pcancel()
	close(release)

	<-prodDone
	<-waiterDone

	require.NoError(t, waiterErr, "a live waiter must not receive the producer's context error")
	require.Equal(t, "fresh", waiterVal)
	require.Equal(t, int32(2), calls.Load(), "the waiter must retry the lookup with its own context")
}

// Test_Lookup_huge_ttl_never_expires verifies that an extreme TTL saturates
// instead of overflowing into an already-expired deadline.
func Test_Lookup_huge_ttl_never_expires(t *testing.T) {
	t.Parallel()

	var i int

	lookupFn := func(_ context.Context, _ string) (any, error) {
		i++

		return i, nil
	}

	c := New(lookupFn, Config{Size: 2, TTL: time.Duration(math.MaxInt64)})

	val, err := c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, 1, val)

	val, err = c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, 1, val, "a huge ttl must not overflow into an expired deadline")
	require.Equal(t, 1, i)
}

// Test_Lookup_unhashable_key_does_not_wedge verifies that the panic caused by
// an unhashable key (allowed by interface-typed keys) does not leak the cache
// lock: the cache must remain fully usable afterwards.
func Test_Lookup_unhashable_key_does_not_wedge(t *testing.T) {
	t.Parallel()

	lookupFn := func(_ context.Context, key any) (any, error) {
		return key, nil
	}

	c := New(lookupFn, Config{Size: 2, TTL: time.Minute})

	require.Panics(t, func() {
		_, _ = c.Lookup(t.Context(), []int{1}) // unhashable key panics in the map access
	})

	done := make(chan struct{})

	go func() {
		defer close(done)

		c.Remove("gone")

		v, err := c.Lookup(context.Background(), "alpha")
		assert.NoError(t, err) // assert (not require) must be used off the test goroutine
		assert.Equal(t, "alpha", v)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("cache wedged after unhashable-key panic")
	}
}

// Test_Lookup_value_with_error_passthrough pins the contract that a non-nil
// value returned alongside a non-nil error is passed through to the producer
// and shared with coalesced waiters as-is.
func Test_Lookup_value_with_error_passthrough(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	errBoom := errors.New("boom")

	lookupFn := func(_ context.Context, _ string) (any, error) {
		<-release // hold the in-flight lookup open

		return "partial", errBoom
	}

	c := New(lookupFn, Config{Size: 2, TTL: time.Minute})

	prodVal := make(chan any, 1)

	go func() {
		v, err := c.Lookup(context.Background(), "alpha")
		assert.ErrorIs(t, err, errBoom) // assert (not require) must be used off the test goroutine

		prodVal <- v
	}()

	// Wait until the producer has registered the in-flight entry.
	require.Eventually(t, func() bool {
		return inFlight(c, "alpha")
	}, time.Second, time.Millisecond)

	waitVal := make(chan any, 1)

	go func() {
		v, err := c.Lookup(context.Background(), "alpha")
		assert.ErrorIs(t, err, errBoom) // assert (not require) must be used off the test goroutine

		waitVal <- v
	}()

	waitForParkedLookupWaiter(t, "value_with_error_passthrough")
	close(release)

	require.Equal(t, "partial", <-prodVal, "the value alongside an error must reach the producer")
	require.Equal(t, "partial", <-waitVal, "the value alongside an error must be shared with waiters")
}

// Test_Lookup_precanceled_context verifies that a fresh cached value is
// served regardless of context state, while a miss never starts an external
// lookup with an already-ended context.
func Test_Lookup_precanceled_context(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	lookupFn := func(_ context.Context, key string) (any, error) {
		calls.Add(1)

		return key, nil
	}

	c := New(lookupFn, Config{Size: 2, TTL: time.Minute})

	// Populate the cache with a live context.
	v, err := c.Lookup(t.Context(), "alpha")
	require.NoError(t, err)
	require.Equal(t, "alpha", v)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// A fresh cached value is served regardless of context state.
	v, err = c.Lookup(ctx, "alpha")
	require.NoError(t, err)
	require.Equal(t, "alpha", v)

	// A miss must not start an external lookup with an ended context.
	v, err = c.Lookup(ctx, "beta")
	require.ErrorIs(t, err, ErrLookupAborted)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, v)
	require.Equal(t, int32(1), calls.Load(), "no lookup must start with an ended context")
}

// Test_Lookup_producer_success_with_dead_context_cached verifies that a
// lookup succeeding despite its caller's context ending mid-flight is still
// cached: success is success.
func Test_Lookup_producer_success_with_dead_context_cached(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	var calls atomic.Int32

	lookupFn := func(_ context.Context, _ string) (any, error) {
		calls.Add(1)
		<-release // hold the in-flight lookup open

		return "value", nil
	}

	c := New(lookupFn, Config{Size: 2, TTL: time.Minute})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		defer close(done)

		v, err := c.Lookup(ctx, "alpha")
		assert.NoError(t, err) // assert (not require) must be used off the test goroutine
		assert.Equal(t, "value", v)
	}()

	// Wait until the producer has registered the in-flight entry.
	require.Eventually(t, func() bool {
		return inFlight(c, "alpha")
	}, time.Second, time.Millisecond)

	// Cancel the producer's context, then let the lookup succeed anyway.
	cancel()
	close(release)
	<-done

	v, err := c.Lookup(t.Context(), "alpha")
	require.NoError(t, err)
	require.Equal(t, "value", v, "a successful result must be cached even with a dead context")
	require.Equal(t, int32(1), calls.Load())
}

// Test_Lookup_genuine_error_with_dead_context_shared verifies that a genuine
// (non-context) lookup error is shared with coalesced waiters even when the
// producer's context happens to be dead, instead of being reclassified as
// context-induced and silently retried.
func Test_Lookup_genuine_error_with_dead_context_shared(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	errUpstream := errors.New("upstream failure")

	var calls atomic.Int32

	lookupFn := func(_ context.Context, _ string) (any, error) {
		calls.Add(1)
		<-release // hold the in-flight lookup open until canceled

		return nil, errUpstream // genuine error, unrelated to the context
	}

	c := New(lookupFn, Config{Size: 2, TTL: time.Minute})

	pctx, pcancel := context.WithCancel(context.Background())
	prodDone := make(chan struct{})

	go func() {
		defer close(prodDone)

		_, err := c.Lookup(pctx, "alpha")
		assert.ErrorIs(t, err, errUpstream) // assert (not require) must be used off the test goroutine
	}()

	// Wait until the producer has registered the in-flight entry.
	require.Eventually(t, func() bool {
		return inFlight(c, "alpha")
	}, time.Second, time.Millisecond)

	var waiterErr error

	waiterDone := make(chan struct{})

	go func() {
		defer close(waiterDone)

		_, waiterErr = c.Lookup(context.Background(), "alpha")
	}()

	waitForParkedLookupWaiter(t, "genuine_error_with_dead_context_shared")

	// Cancel the producer's context, then fail the lookup with a genuine
	// (non-context) error: it must be shared, not silently retried.
	pcancel()
	close(release)

	<-prodDone
	<-waiterDone

	require.ErrorIs(t, waiterErr, errUpstream)
	require.Equal(t, int32(1), calls.Load(), "a genuine error must be shared, not retried")
}

func Test_Lookup_negative_ttl(t *testing.T) {
	t.Parallel()

	var i int

	lookupFn := func(_ context.Context, _ string) (any, error) {
		i++

		return i, nil
	}

	c := New(lookupFn, Config{Size: 2, TTL: -time.Second})

	v, err := c.Lookup(t.Context(), "alpha")
	require.NoError(t, err)
	require.Equal(t, 1, v)

	v, err = c.Lookup(t.Context(), "alpha")
	require.NoError(t, err)
	require.Equal(t, 2, v, "a negative ttl must disable caching")
	require.Equal(t, 2, i)
}

// Test_Lookup_with_ttl_func verifies that a TTL function set via WithTTLFunc
// overrides the cache-wide TTL for entries where it returns a positive
// duration and falls back to the default otherwise.
func Test_Lookup_with_ttl_func(t *testing.T) {
	t.Parallel()

	var i int

	lookupFn := func(_ context.Context, _ string) (any, error) {
		i++

		return i, nil
	}

	ttlFn := func(key string, _ any) time.Duration {
		if key == "short.example.com" {
			return 500 * time.Millisecond
		}

		return 0 // fall back to the cache-wide TTL
	}

	c := New(lookupFn, Config{Size: 4, TTL: 1 * time.Minute}, WithTTLFunc(ttlFn))

	// Both entries are cached: the first with the per-entry TTL, the second
	// with the cache-wide default.
	v, err := c.Lookup(t.Context(), "short.example.com")
	require.NoError(t, err)
	require.Equal(t, 1, v)

	v, err = c.Lookup(t.Context(), "long.example.com")
	require.NoError(t, err)
	require.Equal(t, 2, v)

	v, err = c.Lookup(t.Context(), "short.example.com")
	require.NoError(t, err)
	require.Equal(t, 1, v, "the short entry must be cached within its per-entry TTL")

	time.Sleep(600 * time.Millisecond)

	// The per-entry TTL expired: the short entry is refreshed, while the
	// default-TTL entry is still served from cache.
	v, err = c.Lookup(t.Context(), "short.example.com")
	require.NoError(t, err)
	require.Equal(t, 3, v, "the short entry must be refreshed after its per-entry TTL")

	v, err = c.Lookup(t.Context(), "long.example.com")
	require.NoError(t, err)
	require.Equal(t, 2, v, "the default-TTL entry must still be cached")
	require.Equal(t, 3, i)
}

// Test_Lookup_with_ttl_func_negative verifies that a negative ttlFn result
// falls back to the cache-wide TTL instead of storing an expired entry.
func Test_Lookup_with_ttl_func_negative(t *testing.T) {
	t.Parallel()

	var i int

	lookupFn := func(_ context.Context, _ string) (any, error) {
		i++

		return i, nil
	}

	ttlFn := func(_ string, _ any) time.Duration {
		return -1 * time.Second
	}

	c := New(lookupFn, Config{Size: 2, TTL: 1 * time.Minute}, WithTTLFunc(ttlFn))

	v, err := c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, 1, v)

	v, err = c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, 1, v, "a negative ttlFn result must fall back to the cache-wide TTL")
	require.Equal(t, 1, i)
}

// Test_Lookup_with_ttl_func_value_driven verifies that ttlFn receives the
// actual value returned by the lookup and can derive the TTL from it (the
// documented use case for data that carries its own freshness).
func Test_Lookup_with_ttl_func_value_driven(t *testing.T) {
	t.Parallel()

	var i int

	lookupFn := func(_ context.Context, key string) (any, error) {
		i++

		if key == "cached.example.com" {
			return 1 * time.Minute, nil // the value carries its own TTL
		}

		return time.Duration(0), nil
	}

	ttlFn := func(_ string, v any) time.Duration {
		return v.(time.Duration) //nolint:forcetypeassert
	}

	// The cache-wide ttl is zero: only the value-derived TTL can cache.
	c := New(lookupFn, Config{Size: 4, TTL: 0}, WithTTLFunc(ttlFn))

	v, err := c.Lookup(t.Context(), "cached.example.com")
	require.NoError(t, err)
	require.Equal(t, 1*time.Minute, v)

	v, err = c.Lookup(t.Context(), "cached.example.com")
	require.NoError(t, err)
	require.Equal(t, 1*time.Minute, v, "the value-derived TTL must cache the entry")
	require.Equal(t, 1, i)

	_, err = c.Lookup(t.Context(), "uncached.example.com")
	require.NoError(t, err)

	_, err = c.Lookup(t.Context(), "uncached.example.com")
	require.NoError(t, err)
	require.Equal(t, 3, i, "a zero value-derived TTL must fall back to the (disabled) cache-wide TTL")
}

// Test_Lookup_rejects_a_non_self_equal_key pins that a key that is not equal to
// itself (any key that is or contains a NaN) is rejected outright.
//
// Such a key hashes to a map slot that no subsequent lookup can ever find again.
// Accepting it defeats the cache AND the single flight — every call goes to the
// upstream — while leaking one flight record per call, unreachable by Remove,
// Reset or PurgeExpired. A cache that silently stops caching and stops coalescing
// is a thundering-herd amplifier; rejecting the key says so at the first call.
func Test_Lookup_rejects_a_non_self_equal_key(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	c := New(func(_ context.Context, _ float64) (string, error) {
		calls.Add(1)

		return "value", nil
	}, Config{Size: 4, TTL: time.Hour})

	for range 100 {
		v, err := c.Lookup(t.Context(), math.NaN())
		require.ErrorIs(t, err, ErrInvalidKey)
		require.Empty(t, v)
	}

	require.Zero(t, calls.Load(), "a rejected key must never reach the lookup function")
	require.Zero(t, c.Len(), "a rejected key must leave nothing behind: no entry, no flight")

	c.mux.RLock()
	require.Empty(t, c.keymap)
	require.Empty(t, c.flights, "a non-self-equal key would leak one unreachable flight record per call")
	c.mux.RUnlock()

	// A struct key with a NaN float field is the realistic path here, and it is
	// just as unusable.
	type key struct {
		host  string
		score float64
	}

	sc := New(func(_ context.Context, _ key) (string, error) {
		calls.Add(1)

		return "value", nil
	}, Config{Size: 4, TTL: time.Hour})

	_, err := sc.Lookup(t.Context(), key{host: "example.com", score: math.NaN()})
	require.ErrorIs(t, err, ErrInvalidKey)
	require.Zero(t, sc.Len())

	// A self-equal key of the same types is unaffected.
	v, err := sc.Lookup(t.Context(), key{host: "example.com", score: 1.5})
	require.NoError(t, err)
	require.Equal(t, "value", v)

	requireConsistentAccounting(t, sc)
}

// Test_Lookup_stale_value_is_not_served_to_a_dead_context pins the distinction the
// package doc draws: a FRESH cached value is served regardless of context state,
// but a stale one is not. Serving stale requires attempting a refresh, and no
// lookup is ever started with an already-ended context, so such a caller gets
// ErrLookupAborted rather than the stale value.
func Test_Lookup_stale_value_is_not_served_to_a_dead_context(t *testing.T) {
	t.Parallel()

	var down atomic.Bool

	c := New(func(_ context.Context, _ string) (string, error) {
		if down.Load() {
			return "", errors.New("upstream down")
		}

		return "GOOD", nil
	}, Config{Size: 2, TTL: time.Millisecond, MaxStaleOnFailure: time.Hour})

	_, err := c.Lookup(t.Context(), "k")
	require.NoError(t, err)

	time.Sleep(5 * time.Millisecond) // let the TTL lapse

	down.Store(true)

	// A live context revives the value, which is now cached as a stale entry.
	v, err := c.Lookup(t.Context(), "k")
	require.NoError(t, err)
	require.Equal(t, "GOOD", v)

	dead, cancel := context.WithCancel(context.Background())
	cancel()

	v, err = c.Lookup(dead, "k")
	require.ErrorIs(t, err, ErrLookupAborted, "a stale value must not be served to an already-ended context")
	require.Empty(t, v)

	// The control: a FRESH value IS served to the same dead context.
	down.Store(false)

	fc := New(func(_ context.Context, _ string) (string, error) {
		return "FRESH", nil
	}, Config{Size: 2, TTL: time.Hour})

	_, err = fc.Lookup(t.Context(), "k")
	require.NoError(t, err)

	v, err = fc.Lookup(dead, "k")
	require.NoError(t, err, "a fresh cached value must be served regardless of context state")
	require.Equal(t, "FRESH", v)
}

// Test_Lookup_zero_ttl_retains_a_value_for_stale_if_error pins the qualifier on
// "a TTL <= 0 serves no value from the cache": with stale-if-error enabled the
// value IS retained, and IS served with a nil error after a failed refresh.
func Test_Lookup_zero_ttl_retains_a_value_for_stale_if_error(t *testing.T) {
	t.Parallel()

	for _, cfg := range []Config{
		{Size: 2, TTL: 0, MaxStaleOnFailure: time.Hour},
		{Size: 2, TTL: 0, MaxStale: time.Hour},
		{Size: 2, TTL: -time.Hour, MaxStaleOnFailure: time.Hour},
	} {
		var down atomic.Bool

		c := New(func(_ context.Context, _ string) (string, error) {
			if down.Load() {
				return "", errors.New("upstream down")
			}

			return "GOOD", nil
		}, cfg)

		_, err := c.Lookup(t.Context(), "k")
		require.NoError(t, err)

		down.Store(true)

		v, err := c.Lookup(t.Context(), "k")
		require.NoErrorf(t, err, "a TTL <= 0 must still retain the value for stale-if-error (cfg %+v)", cfg)
		require.Equal(t, "GOOD", v)
	}

	// The control: with no stale window, a TTL <= 0 really does serve nothing.
	var down atomic.Bool

	c := New(func(_ context.Context, _ string) (string, error) {
		if down.Load() {
			return "", errors.New("upstream down")
		}

		return "GOOD", nil
	}, Config{Size: 2, TTL: 0})

	_, err := c.Lookup(t.Context(), "k")
	require.NoError(t, err)

	down.Store(true)

	_, err = c.Lookup(t.Context(), "k")
	require.Error(t, err, "with no stale window, a TTL <= 0 caches nothing at all")
}

// Test_Lookup_a_panicking_context_does_not_leak_the_write_lock pins that a panic out of
// the caller's context cannot take the whole cache with it.
//
// ctx.Err() is code this package does not own, and a nil context makes it panic. If it
// were called under the exclusive write lock, the panic would unwind past the unlock and
// leave the lock held forever: the caller survives (any HTTP server recovers panics) and
// every later caller of the cache blocks for good, including a cache hit.
func Test_Lookup_a_panicking_context_does_not_leak_the_write_lock(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, Config{Size: 4, TTL: time.Minute})

	var nilCtx context.Context

	func() {
		defer func() {
			require.NotNil(t, recover(), "a nil context must panic, or this test proves nothing")
		}()

		_, _ = c.Lookup(nilCtx, "example.com") // a nil context is exactly the point
	}()

	// The cache must still be usable by everybody else.
	done := make(chan struct{})

	go func() {
		defer close(done)

		_ = c.Len() // the read lock

		_, _ = c.Lookup(context.Background(), "example.com") // the write lock
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the cache is wedged: the panic unwound while holding the write lock")
	}
}

// lockingError is an application error whose Is method takes an application lock — an
// error type with a mutable field guarded by a mutex is ordinary Go. errors.Is calls
// it while walking the chain.
type lockingError struct {
	mux    *sync.Mutex
	inside chan struct{} // closed once Is has been entered
	once   sync.Once
}

func (e *lockingError) Error() string { return "upstream failed" }

func (e *lockingError) Is(_ error) bool {
	e.once.Do(func() { close(e.inside) })

	e.mux.Lock()
	defer e.mux.Unlock()

	return false
}

// Test_Lookup_the_error_chain_is_walked_outside_the_write_lock pins the rule that code
// this package does not own is never run while holding the lock.
//
// Deciding whether a failure was context-induced means calling ctx.Err() and walking the
// error chain with errors.Is, which invokes the CALLER's Unwrap and Is methods. Under
// the exclusive write lock, an Is that blocks on a lock the caller also takes around a
// call into this cache deadlocks the two against each other, and the whole cache wedges
// for good.
func Test_Lookup_the_error_chain_is_walked_outside_the_write_lock(t *testing.T) {
	t.Parallel()

	var appMux sync.Mutex // the application's own lock

	failure := &lockingError{mux: &appMux, inside: make(chan struct{})}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := New(func(_ context.Context, _ string) (string, error) {
		// End the producer's OWN context: the only path on which the error chain is
		// walked at all (a nil ctx error short-circuits errors.Is).
		cancel()

		return "", failure
	}, Config{Size: 8, TTL: time.Minute})

	appMux.Lock() // the application holds its lock FIRST

	done := make(chan struct{})

	go func() {
		defer close(done)

		_, _ = c.Lookup(ctx, "key") // -> publish -> errors.Is -> failure.Is -> wants appMux
	}()

	<-failure.inside // the error chain is being walked right now

	// If that walk is happening under the write lock, this call cannot complete: it needs
	// the read lock, which the producer's write lock excludes, and the producer cannot
	// finish until appMux is free, which this call is holding. Deadlock.
	wedged := make(chan struct{})

	go func() {
		defer close(wedged)
		defer appMux.Unlock()

		_ = c.Len()
	}()

	select {
	case <-wedged:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the cache is wedged: the caller's error chain was walked under the write lock")
	}

	<-done
}

// Test_Lookup_the_ttl_func_runs_outside_the_write_lock pins the same rule for the TTL
// function: code this package does not own is never run while holding the lock.
//
// A ttlFn that takes an application lock is an ordinary thing to write (reading a config
// field, consulting a rate limiter). Run under the exclusive write lock, it deadlocks
// against any goroutine that holds that lock and calls into the cache, and the whole
// cache wedges for good.
func Test_Lookup_the_ttl_func_runs_outside_the_write_lock(t *testing.T) {
	t.Parallel()

	var appMux sync.Mutex // the application's own lock

	inTTLFn := make(chan struct{})

	c := New(func(_ context.Context, _ string) (string, error) {
		return "v", nil
	}, Config{Size: 8, TTL: time.Minute},
		WithTTLFunc(func(_ string, _ string) time.Duration {
			close(inTTLFn)

			appMux.Lock() // must not be reached while holding the cache's write lock
			defer appMux.Unlock()

			return time.Minute
		}))

	appMux.Lock() // the application holds its lock FIRST

	done := make(chan struct{})

	go func() {
		defer close(done)

		_, _ = c.Lookup(t.Context(), "key") // -> ttlFn -> wants appMux
	}()

	<-inTTLFn // ttlFn is running right now

	// If ttlFn holds the write lock, this cannot complete: it needs the read lock, which
	// the write lock excludes, and ttlFn cannot return until appMux is free, which this
	// call is holding. Deadlock.
	wedged := make(chan struct{})

	go func() {
		defer close(wedged)
		defer appMux.Unlock()

		_ = c.Len()
	}()

	select {
	case <-wedged:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the cache is wedged: the TTL function was run under the write lock")
	}

	<-done
}
