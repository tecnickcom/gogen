package sfcache

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	got = New[string](nil, 0, 1*time.Second)
	require.Equal(t, 1, got.size)
}

func Test_Len(t *testing.T) {
	t.Parallel()

	c := New[string](nil, 3, 1*time.Second)

	c.keymap = map[string]*entry{
		"example.com": {
			expireAt: time.Now().UTC().UnixNano(),
		},
		"example.net": {
			expireAt: time.Now().UTC().UnixNano(),
		},
	}
	require.Equal(t, 2, c.Len())
}

func Test_Reset(t *testing.T) {
	t.Parallel()

	c := New[string](nil, 1, 1*time.Second)

	c.keymap = map[string]*entry{
		"example.com": {
			expireAt: time.Now().UTC().UnixNano(),
		},
	}

	c.Reset()

	require.Empty(t, c.keymap)
}

func Test_Remove(t *testing.T) {
	t.Parallel()

	c := New[string](nil, 3, 1*time.Second)

	c.keymap = map[string]*entry{
		"example.com": {
			expireAt: time.Now().UTC().UnixNano(),
		},
		"example.net": {
			expireAt: time.Now().UTC().UnixNano(),
		},
		"example.org": {
			expireAt: time.Now().UTC().UnixNano(),
		},
	}

	c.Remove("example.net")

	require.Equal(t, 2, c.Len())
	require.Contains(t, c.keymap, "example.com")
	require.Contains(t, c.keymap, "example.org")
}

func Test_evict_expired(t *testing.T) {
	t.Parallel()

	r := New[string](nil, 3, 1*time.Minute)

	r.keymap = map[string]*entry{
		"example.com": {
			expireAt: time.Now().UTC().Add(-2 * time.Second).UnixNano(),
		},
		"example.org": {
			expireAt: time.Now().UTC().Add(11 * time.Second).UnixNano(),
		},
		"example.net": {
			expireAt: time.Now().UTC().Add(13 * time.Second).UnixNano(),
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

	c := New[string](nil, 3, 1*time.Second)

	c.keymap = map[string]*entry{
		"example.com": {
			expireAt: time.Now().UTC().Add(11 * time.Second).UnixNano(),
		},
		"example.org": {
			expireAt: time.Now().UTC().Add(7 * time.Second).UnixNano(),
		},
		"example.net": {
			expireAt: time.Now().UTC().Add(13 * time.Second).UnixNano(),
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

	c := New[string](nil, 2, 10*time.Second)

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

	c.set("example.org", nil, nil, wait)

	val, err = c.Lookup(ctx, "example.org")
	require.Error(t, err)
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

	time.Sleep(20 * time.Millisecond) // let the waiter block on item.wait
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

	var i int

	lookupFn := func(_ context.Context, _ string) (any, error) {
		time.Sleep(300 * time.Millisecond) // simulate slow lookup

		i++
		ip := fmt.Sprintf("192.0.2.%d", i)

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

	var i int

	lookupFn := func(_ context.Context, _ string) (any, error) {
		time.Sleep(300 * time.Millisecond) // simulate slow lookup

		i++

		return nil, fmt.Errorf("mock error: %d", i)
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

	c := New(lookupFn, 2, 200*time.Millisecond)

	// cache miss
	val, err := c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, "192.0.2.1", val)

	// cache hit: a fresh entry with a sub-second TTL must not be already expired
	val, err = c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, "192.0.2.1", val)
	require.Equal(t, 1, i)

	time.Sleep(250 * time.Millisecond)

	// cache expired
	val, err = c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, "192.0.2.2", val)
	require.Equal(t, 2, i)
}

func Test_evict_skips_inflight_placeholder(t *testing.T) {
	t.Parallel()

	c := New[string](nil, 2, 1*time.Minute)

	wait := make(chan struct{})

	c.keymap = map[string]*entry{
		"inflight.example.com": {
			wait: wait,
		},
		"cached.example.com": {
			expireAt: time.Now().UTC().Add(30 * time.Second).UnixNano(),
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

	time.Sleep(20 * time.Millisecond) // let the duplicate block on item.wait
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

	time.Sleep(20 * time.Millisecond) // let the waiter block on item.wait
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
	c.keymap["alpha"] = &entry{wait: wait}
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
	c.keymap["alpha"] = &entry{wait: wait1}
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
	c.keymap["alpha"] = &entry{wait: wait2}
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

	require.Eventually(t, func() bool {
		buf := make([]byte, 1<<22)
		stacks := string(buf[:runtime.Stack(buf, true)])

		for g := range strings.SplitSeq(stacks, "\n\n") {
			if strings.Contains(g, marker) &&
				strings.Contains(g, "[select") &&
				strings.Contains(g, ").Lookup(") {
				return true
			}
		}

		return false
	}, 5*time.Second, time.Millisecond)
}
