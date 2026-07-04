package sfcache

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nopLookupFn is a placeholder lookup function for tests that never invoke it.
func nopLookupFn(_ context.Context, _ string) (any, error) {
	return nil, nil //nolint:nilnil
}

func TestNew(t *testing.T) {
	t.Parallel()

	testLookup := func(_ context.Context, key string) (any, error) {
		return key, nil
	}

	got := New(testLookup, 3, 5*time.Second)
	require.NotNil(t, got)

	require.NotNil(t, got.lookupFn)
	require.NotNil(t, got.mux)

	require.Equal(t, 3, got.size)
	require.Equal(t, 5*time.Second, got.ttl)

	require.NotNil(t, got.keymap)
	require.Empty(t, got.keymap)

	got = New(testLookup, 0, 1*time.Second)
	require.Equal(t, 1, got.size)

	// A nil lookupFn is replaced by a default function that always fails.
	nilfn := New[string, any](nil, 1, 1*time.Second)

	val, err := nilfn.Lookup(t.Context(), "example.com")
	require.ErrorIs(t, err, ErrNilLookupFunc)
	require.Nil(t, val)
}

func Test_Len(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, 3, 1*time.Second)

	c.keymap = map[string]*entry[any]{
		"example.com": {
			expireAt: time.Now(),
		},
		"example.net": {
			expireAt: time.Now(),
		},
	}
	require.Equal(t, 2, c.Len())
}

func Test_Reset(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, 1, 1*time.Second)

	c.keymap = map[string]*entry[any]{
		"example.com": {
			expireAt: time.Now(),
		},
	}

	c.Reset()

	require.Empty(t, c.keymap)
}

func Test_Remove(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, 3, 1*time.Second)

	c.keymap = map[string]*entry[any]{
		"example.com": {
			expireAt: time.Now(),
		},
		"example.net": {
			expireAt: time.Now(),
		},
		"example.org": {
			expireAt: time.Now(),
		},
	}

	c.Remove("example.net")

	require.Equal(t, 2, c.Len())
	require.Contains(t, c.keymap, "example.com")
	require.Contains(t, c.keymap, "example.org")
}

func Test_evict_expired(t *testing.T) {
	t.Parallel()

	r := New(nopLookupFn, 3, 1*time.Minute)

	r.keymap = map[string]*entry[any]{
		"example.com": {
			expireAt: time.Now().Add(-2 * time.Second),
		},
		"example.org": {
			expireAt: time.Now().Add(11 * time.Second),
		},
		"example.net": {
			expireAt: time.Now().Add(13 * time.Second),
		},
	}

	require.Equal(t, 3, r.Len())

	r.evict()

	require.Equal(t, 2, r.Len())
	require.Contains(t, r.keymap, "example.org")
	require.Contains(t, r.keymap, "example.net")
}

func Test_evict_oldest(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, 3, 1*time.Second)

	c.keymap = map[string]*entry[any]{
		"example.com": {
			expireAt: time.Now().Add(11 * time.Second),
		},
		"example.org": {
			expireAt: time.Now().Add(7 * time.Second),
		},
		"example.net": {
			expireAt: time.Now().Add(13 * time.Second),
		},
	}

	c.evict()

	require.Equal(t, 2, c.Len())
	require.Contains(t, c.keymap, "example.com")
	require.Contains(t, c.keymap, "example.net")
}

/*
NOTE:
The IP blocks 192.0.2.0/24 (TEST-NET-1), 198.51.100.0/24 (TEST-NET-2),
and 203.0.113.0/24 (TEST-NET-3) are provided for use in documentation.
*/

func Test_set(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, 2, 10*time.Second)

	c.set("example.com", []string{"192.0.2.1"}, nil, nil)
	time.Sleep(1 * time.Second)
	c.set("example.org", []string{"192.0.2.2", "198.51.100.2"}, nil, nil)

	require.Equal(t, 2, c.Len())
	require.Contains(t, c.keymap, "example.com")
	require.Contains(t, c.keymap, "example.org")

	c.set("example.net", []string{"192.0.2.3", "198.51.100.3", "203.0.113.3"}, nil, nil)

	require.Equal(t, 2, c.Len())
	require.Contains(t, c.keymap, "example.org")
	require.Contains(t, c.keymap, "example.net")

	c.set("example.net", []string{"198.51.100.4"}, nil, nil)

	require.Equal(t, 2, c.Len())
	require.Contains(t, c.keymap, "example.org")
	require.Contains(t, c.keymap, "example.net")
	require.Equal(t, []string{"198.51.100.4"}, c.keymap["example.net"].val)
}

func Test_Lookup(t *testing.T) {
	t.Parallel()

	var i int

	lookupFn := func(_ context.Context, _ string) (any, error) {
		i++

		ip := fmt.Sprintf("192.0.2.%d", i)

		return []string{ip}, nil
	}

	c := New(lookupFn, 1, 1*time.Second)

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
	wait := make(chan struct{})

	c.mux.Lock()
	c.set("example.org", nil, nil, wait)
	c.mux.Unlock()

	go func() {
		time.Sleep(5 * time.Millisecond)
		c.Remove("example.org")
		close(wait)
	}()

	val, err = c.Lookup(t.Context(), "example.org")
	require.NoError(t, err)
	require.Equal(t, []string{"192.0.2.4"}, val)

	// context expired on duplicate lookup
	wait = make(chan struct{})

	c.mux.Lock()
	c.set("example.org", nil, nil, wait)
	c.mux.Unlock()

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

	c := New(lookupFn, 4, time.Minute)

	prodDone := make(chan struct{})

	go func() {
		defer close(prodDone)

		v, err := c.Lookup(context.Background(), "example.org")
		assert.NoError(t, err) // assert (not require) must be used off the test goroutine
		assert.Equal(t, []string{"192.0.2.1"}, v)
	}()

	// Wait until the producer has registered the in-flight entry.
	require.Eventually(t, func() bool {
		c.mux.RLock()
		defer c.mux.RUnlock()

		it, ok := c.keymap["example.org"]

		return ok && it.wait != nil
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

	c := New(lookupFn, 2, 0)
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

	c := New(lookupFn, 2, 0)
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

	c := New(lookupFn, 2, 10*time.Second)

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

	c := New(lookupFn, 2, 10*time.Second)

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

	c := New(lookupFn, 2, 10*time.Second)

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

	c := New(lookupFn, 2, 500*time.Millisecond)

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

func Test_evict_skips_inflight_placeholder(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, 2, 1*time.Minute)

	wait := make(chan struct{})

	c.keymap = map[string]*entry[any]{
		"inflight.example.com": {
			wait: wait,
		},
		"cached.example.com": {
			expireAt: time.Now().Add(30 * time.Second),
		},
	}

	c.evict()

	require.Contains(t, c.keymap, "inflight.example.com", "in-flight placeholders must not be evicted")
	require.NotContains(t, c.keymap, "cached.example.com")

	// With only in-flight placeholders left, evict must be a no-op.
	c.evict()

	require.Contains(t, c.keymap, "inflight.example.com")

	close(wait)
}

func Test_Lookup_capacity_preserves_inflight(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	var (
		mu    sync.Mutex
		calls = map[string]int{}
	)

	lookupFn := func(_ context.Context, key string) (any, error) {
		mu.Lock()
		calls[key]++
		mu.Unlock()

		if key == "slow.example.com" {
			<-release // hold the in-flight lookup open
		}

		return key, nil
	}

	c := New(lookupFn, 1, 1*time.Minute)

	prodDone := make(chan struct{})

	go func() {
		defer close(prodDone)

		v, err := c.Lookup(context.Background(), "slow.example.com")
		assert.NoError(t, err) // assert (not require) must be used off the test goroutine
		assert.Equal(t, "slow.example.com", v)
	}()

	// Wait until the producer has registered the in-flight entry.
	require.Eventually(t, func() bool {
		c.mux.RLock()
		defer c.mux.RUnlock()

		it, ok := c.keymap["slow.example.com"]

		return ok && it.wait != nil
	}, time.Second, time.Millisecond)

	// Fill the cache beyond capacity with another key:
	// the in-flight placeholder must not be evicted.
	v, err := c.Lookup(t.Context(), "fast.example.com")
	require.NoError(t, err)
	require.Equal(t, "fast.example.com", v)

	c.mux.RLock()
	_, ok := c.keymap["slow.example.com"]
	c.mux.RUnlock()

	require.True(t, ok, "the in-flight placeholder must survive eviction at capacity")

	// A duplicate call must coalesce onto the in-flight lookup.
	dupDone := make(chan struct{})

	go func() {
		defer close(dupDone)

		v, err := c.Lookup(context.Background(), "slow.example.com")
		assert.NoError(t, err) // assert (not require) must be used off the test goroutine
		assert.Equal(t, "slow.example.com", v)
	}()

	waitForParkedLookupWaiter(t, "capacity_preserves_inflight")
	close(release)

	<-prodDone
	<-dupDone

	mu.Lock()
	defer mu.Unlock()

	require.Equal(t, 1, calls["slow.example.com"], "duplicate in-flight lookups must be coalesced")
}

func Test_Lookup_panic_finalizes_placeholder(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	var calls atomic.Int64

	lookupFn := func(_ context.Context, _ string) (any, error) {
		if calls.Add(1) == 1 {
			<-release // hold the in-flight lookup open

			panic("lookup panic")
		}

		return []string{"192.0.2.1"}, nil
	}

	c := New(lookupFn, 4, 1*time.Minute)

	prodDone := make(chan struct{})

	go func() {
		defer close(prodDone)

		defer func() {
			// The panic must propagate to the caller that ran lookupFn.
			assert.NotNil(t, recover()) // assert (not require) must be used off the test goroutine
		}()

		_, _ = c.Lookup(context.Background(), "example.org")
	}()

	// Wait until the producer has registered the in-flight entry.
	require.Eventually(t, func() bool {
		c.mux.RLock()
		defer c.mux.RUnlock()

		it, ok := c.keymap["example.org"]

		return ok && it.wait != nil
	}, time.Second, time.Millisecond)

	type retval struct {
		err error
		val any
	}

	ret := make(chan retval, 1)

	go func() {
		val, err := c.Lookup(context.Background(), "example.org")
		ret <- retval{err, val}
	}()

	waitForParkedLookupWaiter(t, "panic_finalizes_placeholder")
	close(release)

	<-prodDone

	// The waiter must observe a terminal state (placeholder removed) and
	// complete with a fresh lookup instead of busy-spinning forever.
	select {
	case v := <-ret:
		require.NoError(t, v.err)
		require.Equal(t, []string{"192.0.2.1"}, v.val)
	case <-time.After(2 * time.Second):
		t.Fatal("waiter did not terminate after lookupFn panic")
	}

	c.mux.RLock()
	item, ok := c.keymap["example.org"]
	c.mux.RUnlock()

	require.True(t, ok)
	require.Nil(t, item.wait, "no in-flight placeholder must be left behind")
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

	c := New(lookupFn, 2, 0)

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

// Test_Lookup_waiterRecoversWhenEntryRemovedDuringWait drives a waiter through
// the wait-loop branch where the in-flight entry disappears during the wait
// (the state a panicking lookupFn leaves behind): the waiter must observe the
// removed entry, break out of the wait loop, and perform a fresh lookup.
func Test_Lookup_waiterRecoversWhenEntryRemovedDuringWait(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	lookupFn := func(_ context.Context, _ string) (any, error) {
		calls.Add(1)

		return "value", nil
	}

	c := New[string](lookupFn, 3, time.Minute)

	// Install an in-flight placeholder manually so the waiter blocks on it.
	wait := make(chan struct{})

	c.mux.Lock()
	c.keymap["alpha"] = &entry[any]{wait: wait}
	c.mux.Unlock()

	var (
		waiterVal any
		waiterErr error
	)

	waiterDone := make(chan struct{})

	go func() {
		defer close(waiterDone)

		waiterVal, waiterErr = c.Lookup(context.Background(), "alpha")
	}()

	waitForParkedLookupWaiter(t, "waiterRecoversWhenEntryRemovedDuringWait")

	// Remove the entry (as the panic-recovery path does) and wake the waiter:
	// it must observe the removed entry and run a fresh lookup.
	c.mux.Lock()
	delete(c.keymap, "alpha")
	c.mux.Unlock()

	close(wait)

	<-waiterDone
	require.NoError(t, waiterErr)
	require.Equal(t, "value", waiterVal)
	require.Equal(t, int32(1), calls.Load())
}

// Test_Lookup_waiterContinuesWhenEntryUpdatedDuringWait drives a waiter
// through the wait-loop branch where the entry is replaced with a new
// in-flight placeholder during the wait: the waiter must loop again and wait
// on the new placeholder (here until the context is canceled).
func Test_Lookup_waiterContinuesWhenEntryUpdatedDuringWait(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	lookupFn := func(_ context.Context, _ string) (any, error) {
		calls.Add(1)

		return "value", nil
	}

	c := New[string](lookupFn, 3, time.Minute)

	wait1 := make(chan struct{})
	wait2 := make(chan struct{})

	c.mux.Lock()
	c.keymap["alpha"] = &entry[any]{wait: wait1}
	c.mux.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var waiterErr error

	waiterDone := make(chan struct{})

	go func() {
		defer close(waiterDone)

		_, waiterErr = c.Lookup(ctx, "alpha")
	}()

	waitForParkedLookupWaiter(t, "waiterContinuesWhenEntryUpdatedDuringWait")

	// Replace the placeholder with a new in-flight one, then wake the waiter:
	// its parked select is already committed to the wait1 case when the
	// channel closes, so it must re-check, find the new placeholder, and loop
	// again; the context cancellation then unblocks the second wait.
	c.mux.Lock()
	c.keymap["alpha"] = &entry[any]{wait: wait2}
	c.mux.Unlock()

	close(wait1)
	cancel()

	<-waiterDone
	require.ErrorContains(t, waiterErr, "context canceled")
	require.Equal(t, int32(0), calls.Load())
}

// waitForParkedLookupWaiter blocks until a goroutine whose stack contains
// marker is parked in Lookup's wait select, making test interleavings
// deterministic regardless of scheduler load.
func waitForParkedLookupWaiter(t *testing.T, marker string) {
	t.Helper()

	waitForNParkedLookupWaiters(t, marker, 1)
}

// waitForNParkedLookupWaiters blocks until at least n goroutines whose stacks
// contain marker are parked in Lookup's wait select.
func waitForNParkedLookupWaiters(t *testing.T, marker string, n int) {
	t.Helper()

	require.Eventually(t, func() bool {
		parked := 0

		for g := range strings.SplitSeq(allGoroutineStacks(), "\n\n") {
			if strings.Contains(g, marker) &&
				strings.Contains(g, "[select") &&
				strings.Contains(g, ").Lookup(") {
				parked++
			}
		}

		return parked >= n
	}, 5*time.Second, 5*time.Millisecond)
}

// allGoroutineStacks returns the full goroutine dump, growing the buffer
// until it fits, so that parked goroutines are never missed to truncation.
func allGoroutineStacks() string {
	for size := 1 << 20; ; size *= 2 {
		buf := make([]byte, size)

		if n := runtime.Stack(buf, true); n < size {
			return string(buf[:n])
		}
	}
}

// Test_Lookup_removed_entry_single_flight_retry drives two waiters through
// the removed-entry wake-up path: after the placeholder is deleted and its
// channel closed, exactly one waiter must start the replacement lookup and
// the other must coalesce onto it instead of spawning a duplicate flight.
func Test_Lookup_removed_entry_single_flight_retry(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	var calls atomic.Int32

	lookupFn := func(_ context.Context, _ string) (any, error) {
		calls.Add(1)
		<-release // hold the retry lookup open

		return "value", nil
	}

	c := New(lookupFn, 3, time.Minute)

	// Install an in-flight placeholder manually so both waiters block on it.
	wait := make(chan struct{})

	c.mux.Lock()
	c.keymap["alpha"] = &entry[any]{wait: wait}
	c.mux.Unlock()

	results := make(chan error, 2)

	for range 2 {
		go func() {
			val, err := c.Lookup(context.Background(), "alpha")
			if err == nil {
				assert.Equal(t, "value", val) // assert (not require) must be used off the test goroutine
			}

			results <- err
		}()
	}

	waitForNParkedLookupWaiters(t, "removed_entry_single_flight_retry", 2)

	// Remove the entry and wake both waiters: one must become the new
	// producer and the other must coalesce onto its flight.
	c.mux.Lock()
	delete(c.keymap, "alpha")
	c.mux.Unlock()

	close(wait)

	// Wait until the retry flight is registered and the other waiter is
	// parked on its wait channel.
	require.Eventually(t, func() bool {
		c.mux.RLock()
		it, ok := c.keymap["alpha"]
		c.mux.RUnlock()

		return ok && it.wait != nil && calls.Load() == 1
	}, time.Second, time.Millisecond)

	waitForParkedLookupWaiter(t, "removed_entry_single_flight_retry")

	close(release)

	require.NoError(t, <-results)
	require.NoError(t, <-results)
	require.Equal(t, int32(1), calls.Load(), "the removed-entry retry must be single-flight")
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

	c := New(lookupFn, 4, time.Minute)

	pctx, pcancel := context.WithCancel(context.Background())
	prodDone := make(chan struct{})

	go func() {
		defer close(prodDone)

		_, err := c.Lookup(pctx, "example.org")
		assert.ErrorIs(t, err, context.Canceled) // assert (not require) must be used off the test goroutine
	}()

	// Wait until the producer has registered the in-flight entry.
	require.Eventually(t, func() bool {
		c.mux.RLock()
		defer c.mux.RUnlock()

		it, ok := c.keymap["example.org"]

		return ok && it.wait != nil
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

// Test_Remove_during_inflight_prevents_caching verifies that removing a key
// while its lookup is in flight invalidates the flight: the result is still
// returned to its caller, but it is not cached.
func Test_Remove_during_inflight_prevents_caching(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	var calls atomic.Int32

	lookupFn := func(_ context.Context, _ string) (any, error) {
		if calls.Add(1) == 1 {
			<-release // hold the in-flight lookup open

			return "stale", nil
		}

		return "fresh", nil
	}

	c := New(lookupFn, 4, time.Minute)

	prodDone := make(chan struct{})

	go func() {
		defer close(prodDone)

		v, err := c.Lookup(context.Background(), "example.org")
		assert.NoError(t, err) // assert (not require) must be used off the test goroutine
		assert.Equal(t, "stale", v)
	}()

	// Wait until the producer has registered the in-flight entry.
	require.Eventually(t, func() bool {
		c.mux.RLock()
		defer c.mux.RUnlock()

		it, ok := c.keymap["example.org"]

		return ok && it.wait != nil
	}, time.Second, time.Millisecond)

	// Invalidate the key mid-flight, then let the lookup complete.
	c.Remove("example.org")
	close(release)
	<-prodDone

	// The invalidated result must not have been cached:
	// a new call must trigger a fresh lookup.
	v, err := c.Lookup(t.Context(), "example.org")
	require.NoError(t, err)
	require.Equal(t, "fresh", v)
	require.Equal(t, int32(2), calls.Load())
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

	c := New(lookupFn, 2, time.Duration(math.MaxInt64))

	val, err := c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, 1, val)

	val, err = c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, 1, val, "a huge ttl must not overflow into an expired deadline")
	require.Equal(t, 1, i)
}

// Test_Lookup_capacity_overshoot_reclaimed verifies that the capacity
// overshoot caused by more concurrent distinct in-flight keys than `size` is
// reclaimed once the lookups complete, instead of pinning the map at the
// burst high-water mark forever.
func Test_Lookup_capacity_overshoot_reclaimed(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	lookupFn := func(_ context.Context, key string) (any, error) {
		<-release // hold all burst lookups open concurrently

		return key, nil
	}

	c := New(lookupFn, 2, time.Minute)

	const burst = 8

	wg := &sync.WaitGroup{}

	for i := range burst {
		wg.Go(func() {
			_, _ = c.Lookup(context.Background(), fmt.Sprintf("burst%d", i))
		})
	}

	// Wait until all in-flight placeholders exceed the capacity.
	require.Eventually(t, func() bool {
		return c.Len() == burst
	}, 5*time.Second, time.Millisecond)

	close(release)
	wg.Wait()

	// The overshoot must be reclaimed as the lookups publish their results.
	require.LessOrEqual(t, c.Len(), 2, "capacity overshoot must be reclaimed after the burst")

	// And the cache must stay within capacity on subsequent inserts.
	for i := range 5 {
		_, err := c.Lookup(t.Context(), fmt.Sprintf("post%d", i))
		require.NoError(t, err)
		require.LessOrEqual(t, c.Len(), 2)
	}
}

// Test_Lookup_unhashable_key_does_not_wedge verifies that the panic caused by
// an unhashable key (allowed by interface-typed keys) does not leak the cache
// lock: the cache must remain fully usable afterwards.
func Test_Lookup_unhashable_key_does_not_wedge(t *testing.T) {
	t.Parallel()

	lookupFn := func(_ context.Context, key any) (any, error) {
		return key, nil
	}

	c := New(lookupFn, 2, time.Minute)

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

// Test_Reset_during_inflight_stale_flight_guard verifies that a flight
// invalidated by Reset cannot clobber the placeholder of a successor flight
// for the same key when it completes.
func Test_Reset_during_inflight_stale_flight_guard(t *testing.T) {
	t.Parallel()

	relA := make(chan struct{})
	relB := make(chan struct{})

	var calls atomic.Int32

	lookupFn := func(_ context.Context, _ string) (any, error) {
		if calls.Add(1) == 1 {
			<-relA // stale flight A

			return "A", nil
		}

		<-relB // successor flight B

		return "B", nil
	}

	c := New(lookupFn, 4, time.Minute)

	aDone := make(chan struct{})

	go func() {
		defer close(aDone)

		v, err := c.Lookup(context.Background(), "alpha")
		assert.NoError(t, err) // assert (not require) must be used off the test goroutine
		assert.Equal(t, "A", v, "the removed flight's result must still reach its caller")
	}()

	// Wait until flight A has registered its in-flight placeholder.
	require.Eventually(t, func() bool {
		c.mux.RLock()
		defer c.mux.RUnlock()

		it, ok := c.keymap["alpha"]

		return ok && it.wait != nil
	}, time.Second, time.Millisecond)

	// Invalidate flight A mid-flight, then start flight B for the same key.
	c.Reset()

	bDone := make(chan struct{})

	go func() {
		defer close(bDone)

		v, err := c.Lookup(context.Background(), "alpha")
		assert.NoError(t, err) // assert (not require) must be used off the test goroutine
		assert.Equal(t, "B", v)
	}()

	// Wait until flight B has installed its own placeholder.
	require.Eventually(t, func() bool {
		c.mux.RLock()
		defer c.mux.RUnlock()

		it, ok := c.keymap["alpha"]

		return ok && it.wait != nil && calls.Load() == 2
	}, time.Second, time.Millisecond)

	// Complete the stale flight A: it must not clobber B's placeholder.
	close(relA)
	<-aDone

	c.mux.RLock()
	it, ok := c.keymap["alpha"]
	c.mux.RUnlock()

	require.True(t, ok)
	require.NotNil(t, it.wait, "a stale flight must not clobber the successor's placeholder")

	close(relB)
	<-bDone

	v, err := c.Lookup(t.Context(), "alpha")
	require.NoError(t, err)
	require.Equal(t, "B", v, "the successor flight's value must be the cached one")
	require.Equal(t, int32(2), calls.Load())
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

	c := New(lookupFn, 2, time.Minute)

	prodVal := make(chan any, 1)

	go func() {
		v, err := c.Lookup(context.Background(), "alpha")
		assert.ErrorIs(t, err, errBoom) // assert (not require) must be used off the test goroutine

		prodVal <- v
	}()

	// Wait until the producer has registered the in-flight entry.
	require.Eventually(t, func() bool {
		c.mux.RLock()
		defer c.mux.RUnlock()

		it, ok := c.keymap["alpha"]

		return ok && it.wait != nil
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

	c := New(lookupFn, 2, time.Minute)

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

	c := New(lookupFn, 2, time.Minute)

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
		c.mux.RLock()
		defer c.mux.RUnlock()

		it, ok := c.keymap["alpha"]

		return ok && it.wait != nil
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

	c := New(lookupFn, 2, time.Minute)

	pctx, pcancel := context.WithCancel(context.Background())
	prodDone := make(chan struct{})

	go func() {
		defer close(prodDone)

		_, err := c.Lookup(pctx, "alpha")
		assert.ErrorIs(t, err, errUpstream) // assert (not require) must be used off the test goroutine
	}()

	// Wait until the producer has registered the in-flight entry.
	require.Eventually(t, func() bool {
		c.mux.RLock()
		defer c.mux.RUnlock()

		it, ok := c.keymap["alpha"]

		return ok && it.wait != nil
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

func Test_PurgeExpired(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, 4, time.Minute)

	wait := make(chan struct{})

	c.mux.Lock()
	c.keymap = map[string]*entry[any]{
		"expired.example.com": {
			expireAt: time.Now().Add(-time.Second),
		},
		"fresh.example.com": {
			expireAt: time.Now().Add(30 * time.Second),
		},
		"inflight.example.com": {
			wait: wait,
		},
		"stale.example.com": {
			// a revived stale entry: expired, with an open stale window
			staleUntil: time.Now().Add(30 * time.Second),
		},
	}
	c.mux.Unlock()

	require.Equal(t, 2, c.PurgeExpired(), "expired and revived stale entries must be purged")

	require.Equal(t, 2, c.Len())
	require.Contains(t, c.keymap, "fresh.example.com")
	require.Contains(t, c.keymap, "inflight.example.com", "in-flight placeholders must not be purged")

	require.Equal(t, 0, c.PurgeExpired())

	close(wait)
}

// Test_Len_counts_placeholders_and_error_residue pins the documented Len
// semantics: in-flight placeholders and expired error residue are counted.
func Test_Len_counts_placeholders_and_error_residue(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	lookupFn := func(_ context.Context, key string) (any, error) {
		if key == "slow.example.com" {
			<-release // hold the in-flight lookup open

			return key, nil
		}

		return nil, errors.New("mock error")
	}

	c := New(lookupFn, 4, time.Minute)

	done := make(chan struct{})

	go func() {
		defer close(done)

		_, _ = c.Lookup(context.Background(), "slow.example.com")
	}()

	// An in-flight placeholder is counted by Len.
	require.Eventually(t, func() bool {
		return c.Len() == 1
	}, time.Second, time.Millisecond)

	close(release)
	<-done

	// A failed lookup leaves an expired residue entry, also counted by Len.
	_, err := c.Lookup(t.Context(), "failed.example.com")
	require.Error(t, err)
	require.Equal(t, 2, c.Len())
}

// Test_Lookup_evicts_error_residue_first verifies through the public API that
// at capacity the expired residue of a failed lookup is evicted before an
// older but still valid entry.
func Test_Lookup_evicts_error_residue_first(t *testing.T) {
	t.Parallel()

	lookupFn := func(_ context.Context, key string) (any, error) {
		if key == "bad.example.com" {
			return nil, errors.New("mock error")
		}

		return key, nil
	}

	c := New(lookupFn, 2, time.Minute)

	_, err := c.Lookup(t.Context(), "bad.example.com") // leaves an expired residue
	require.Error(t, err)

	v, err := c.Lookup(t.Context(), "good.example.com")
	require.NoError(t, err)
	require.Equal(t, "good.example.com", v)

	// At capacity, the expired residue must be evicted before the valid entry.
	_, err = c.Lookup(t.Context(), "new.example.com")
	require.NoError(t, err)

	c.mux.RLock()
	_, hasGood := c.keymap["good.example.com"]
	_, hasBad := c.keymap["bad.example.com"]
	c.mux.RUnlock()

	require.True(t, hasGood, "the valid entry must survive eviction")
	require.False(t, hasBad, "the expired error residue must be evicted first")
}

func Test_Lookup_negative_ttl(t *testing.T) {
	t.Parallel()

	var i int

	lookupFn := func(_ context.Context, _ string) (any, error) {
		i++

		return i, nil
	}

	c := New(lookupFn, 2, -time.Second)

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

	c := New(lookupFn, 4, 1*time.Minute, WithTTLFunc(ttlFn))

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

	c := New(lookupFn, 4, 0, WithTTLFunc(ttlFn), WithStaleIfError[string, any](1*time.Minute))

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

	c := New(lookupFn, 4, 200*time.Millisecond, WithStaleIfError[string, any](200*time.Millisecond))

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

	c := New(lookupFn, 4, 100*time.Millisecond, WithStaleIfError[string, any](1*time.Minute))

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

	// Wait until the refresh flight has registered its placeholder.
	require.Eventually(t, func() bool {
		c.mux.RLock()
		defer c.mux.RUnlock()

		it, ok := c.keymap["example.com"]

		return ok && it.wait != nil
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

	c := New(lookupFn, 4, 1*time.Minute, WithStaleIfError[string, any](1*time.Minute))

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

	c := New(lookupFn, 4, 100*time.Millisecond, WithStaleIfError[string, any](1*time.Minute))

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

	// Wait until the refresh flight has registered its placeholder.
	require.Eventually(t, func() bool {
		c.mux.RLock()
		defer c.mux.RUnlock()

		it, ok := c.keymap["example.com"]

		return ok && it.wait != nil
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
	c := New(lookupFn, 4, 100*time.Millisecond, WithStaleIfError[string, any](50*time.Millisecond))

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

	// Wait until the refresh flight has registered its placeholder.
	require.Eventually(t, func() bool {
		c.mux.RLock()
		defer c.mux.RUnlock()

		it, ok := c.keymap["example.com"]

		return ok && it.wait != nil
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
	c := New(lookupFn, 4, 200*time.Millisecond, WithStaleIfError[string, any](600*time.Millisecond))

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

	c := New(lookupFn, 2, 1*time.Minute, WithTTLFunc(ttlFn))

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
	c := New(lookupFn, 4, 0, WithTTLFunc(ttlFn))

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

	c := New(lookupFn, 2, 0, WithStaleIfError[string, any](1*time.Minute))

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

	c := New(lookupFn, 1, 100*time.Millisecond, WithStaleIfError[string, any](1*time.Minute))

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

	c := New(lookupFn, 2, 100*time.Millisecond, WithStaleIfError[string, any](1*time.Minute))

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

	c := New(lookupFn, 2, 100*time.Millisecond, WithStaleIfError[string, any](1*time.Minute))

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
