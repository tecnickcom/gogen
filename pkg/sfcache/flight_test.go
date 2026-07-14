// Tests for the in-flight lookup lifecycle.

package sfcache

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Lookup_panic_deregisters_the_flight(t *testing.T) {
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

	c := New(lookupFn, Config{Size: 4, TTL: 1 * time.Minute})

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
		return inFlight(c, "example.org")
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

	waitForParkedLookupWaiter(t, "panic_deregisters_the_flight")
	close(release)

	<-prodDone

	// The waiter must observe a terminal state (no entry, no flight) and
	// complete with a fresh lookup instead of busy-spinning forever.
	select {
	case v := <-ret:
		require.NoError(t, v.err)
		require.Equal(t, []string{"192.0.2.1"}, v.val)
	case <-time.After(2 * time.Second):
		t.Fatal("waiter did not terminate after lookupFn panic")
	}

	c.mux.RLock()
	_, ok := c.keymap["example.org"]
	c.mux.RUnlock()

	require.True(t, ok)
	require.False(t, inFlight(c, "example.org"), "no in-flight lookup must be left behind")
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

	c := New[string](lookupFn, Config{Size: 3, TTL: time.Minute})

	// Register an in-flight lookup manually so the waiter blocks on it.
	fl := seedFlight(c, "alpha")

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

	// Deregister the flight (as the panic-recovery path does) and wake the
	// waiter: it must observe the terminal state and run a fresh lookup.
	// The flight is deregistered before it is finished, so the woken waiter
	// cannot find it again.
	c.mux.Lock()
	delete(c.flights, "alpha")
	c.mux.Unlock()

	fl.finish()

	<-waiterDone
	require.NoError(t, waiterErr)
	require.Equal(t, "value", waiterVal)
	require.Equal(t, int32(1), calls.Load())
}

// Test_Lookup_waiterContinuesWhenEntryUpdatedDuringWait drives a waiter
// through the wait-loop branch where the entry is replaced with a new
// in-flight lookup during the wait: the waiter must loop again and wait
// on the new flight (here until the context is canceled).
func Test_Lookup_waiterContinuesWhenEntryUpdatedDuringWait(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	lookupFn := func(_ context.Context, _ string) (any, error) {
		calls.Add(1)

		return "value", nil
	}

	c := New[string](lookupFn, Config{Size: 3, TTL: time.Minute})

	fl1 := seedFlight(c, "alpha")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var waiterErr error

	waiterDone := make(chan struct{})

	go func() {
		defer close(waiterDone)

		_, waiterErr = c.Lookup(ctx, "alpha")
	}()

	waitForParkedLookupWaiter(t, "waiterContinuesWhenEntryUpdatedDuringWait")

	// Replace the flight with a new one, then wake the waiter: its parked
	// select is already committed to the first flight when the channel closes,
	// so it must re-check, find the new flight, and loop again; the context
	// cancellation then unblocks the second wait.
	seedFlight(c, "alpha")

	fl1.finish()
	cancel()

	<-waiterDone
	require.ErrorContains(t, waiterErr, "context canceled")
	require.Equal(t, int32(0), calls.Load())
}

// Test_Lookup_removed_entry_single_flight_retry drives two waiters through
// the removed-entry wake-up path: after the flight is deregistered and its
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

	c := New(lookupFn, Config{Size: 3, TTL: time.Minute})

	// Register an in-flight lookup manually so both waiters block on it.
	fl := seedFlight(c, "alpha")

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

	// Invalidate the flight: Remove releases both waiters itself. One must become
	// the new producer and the other must coalesce onto its flight.
	c.Remove("alpha")

	requireFinished(t, fl, "Remove must release the waiters it orphans")

	fl.finish() // idempotent: Remove already finished it

	// Wait until the retry flight is registered and the other waiter is
	// parked on its wait channel.
	require.Eventually(t, func() bool {
		return inFlight(c, "alpha") && calls.Load() == 1
	}, time.Second, time.Millisecond)

	waitForParkedLookupWaiter(t, "removed_entry_single_flight_retry")

	close(release)

	require.NoError(t, <-results)
	require.NoError(t, <-results)
	require.Equal(t, int32(1), calls.Load(), "the removed-entry retry must be single-flight")
}

// Test_Lookup_panicking_ttl_func_does_not_deadlock pins the lock ordering that
// fetch's deferred abortFlight depends on. ttlFn runs under the write lock, so
// its panic must unwind through publish's deferred unlock before abortFlight
// re-locks the non-reentrant mutex. Replacing that deferred unlock with an
// explicit one wedges the whole cache, and only this test catches it.
func Test_Lookup_panicking_ttl_func_does_not_deadlock(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	lookupFn := func(_ context.Context, key string) (any, error) {
		if key == "panic.example.com" {
			<-release // hold the flight open so the callers coalesce onto it
		}

		return key, nil
	}

	ttlFn := func(key string, _ any) time.Duration {
		if key == "panic.example.com" {
			panic("mock ttlFn panic")
		}

		return time.Minute
	}

	c := New(lookupFn, Config{Size: 4, TTL: time.Minute}, WithTTLFunc(ttlFn))

	const callers = 8

	var (
		panicked atomic.Int32
		wg       sync.WaitGroup
	)

	for range callers {
		wg.Go(func() {
			defer func() {
				if r := recover(); r != nil {
					panicked.Add(1)
				}
			}()

			_, _ = c.Lookup(context.Background(), "panic.example.com")
		})
	}

	waitForNParkedLookupWaiters(t, "Test_Lookup_panicking_ttl_func", callers-1)
	close(release)

	done := make(chan struct{})

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("a panicking ttlFn deadlocked the cache")
	}

	require.Equal(t, int32(callers), panicked.Load(), "every caller must return by panicking, none may hang")
	require.Equal(t, 0, c.Len(), "the flight must be deregistered")

	// The write lock must not have been left held: the cache is still usable.
	v, err := c.Lookup(t.Context(), "ok.example.com")
	require.NoError(t, err)
	require.Equal(t, "ok.example.com", v)

	requireConsistentAccounting(t, c)
}

// Test_abortFlight_keeps_a_superseded_flight pins that a panicking flight
// only ever drops its OWN flight: when the key was removed and taken over
// by a newer flight, that flight's entry must survive the unwind.
func Test_abortFlight_keeps_a_superseded_flight(t *testing.T) {
	t.Parallel()

	var (
		relA  = make(chan struct{})
		relB  = make(chan struct{})
		calls atomic.Int32
	)

	lookupFn := func(_ context.Context, _ string) (any, error) {
		if calls.Add(1) == 1 {
			<-relA // hold the first flight open, then panic

			panic("mock lookup panic")
		}

		<-relB // the successor is still in flight while the first one unwinds

		return "second", nil
	}

	c := New(lookupFn, Config{Size: 2, TTL: time.Minute})

	first := make(chan struct{})

	go func() {
		defer close(first)

		defer func() { _ = recover() }()

		_, _ = c.Lookup(context.Background(), "example.com")
	}()

	require.Eventually(t, func() bool {
		return inFlight(c, "example.com") && (calls.Load() == 1)
	}, time.Second, time.Millisecond, "the first flight must have registered itself")

	// Remove frees the key while the first flight is still running, so the
	// second lookup below starts a new flight that supersedes it.
	c.Remove("example.com")

	second := make(chan struct{})

	go func() {
		defer close(second)

		v, err := c.Lookup(context.Background(), "example.com")
		assert.NoError(t, err) // assert (not require) must be used off the test goroutine
		assert.Equal(t, "second", v)
	}()

	require.Eventually(t, func() bool {
		return inFlight(c, "example.com") && (calls.Load() == 2)
	}, time.Second, time.Millisecond, "the successor flight must have registered itself")

	// Let the superseded flight panic and unwind WHILE the successor is still in
	// flight: its unwind must not deregister the flight that replaced it, or the
	// successor's result is silently discarded.
	close(relA)
	<-first

	require.True(t, inFlight(c, "example.com"), "a superseded flight must not deregister its successor")

	close(relB)
	<-second

	// The successor's value must have been cached: no further lookup runs.
	before := calls.Load()

	v, err := c.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, "second", v)
	require.Equal(t, before, calls.Load(), "the successor's result must have been cached")

	requireConsistentAccounting(t, c)
}
