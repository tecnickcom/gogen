// Tests for the cache lifecycle: construction and the explicit control methods.

package sfcache

import (
	"context"
	"errors"
	"fmt"
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

	got := New(testLookup, Config{
		Size:              3,
		TTL:               5 * time.Second,
		MaxStale:          7 * time.Second,
		MaxStaleOnFailure: 11 * time.Second,
	})
	require.NotNil(t, got)

	require.NotNil(t, got.lookupFn)

	require.Equal(t, 3, got.size)
	require.Equal(t, 5*time.Second, got.ttl)
	require.Equal(t, 7*time.Second, got.maxStale)
	require.Equal(t, 11*time.Second, got.maxStaleOnFailure)

	require.NotNil(t, got.keymap)
	require.Empty(t, got.keymap)

	// The zero Config is valid: capacity is clamped to 1 and nothing is cached.
	got = New(testLookup, Config{})
	require.Equal(t, 1, got.size)
	require.Equal(t, time.Duration(0), got.ttl)

	// Any Size <= 0 is clamped to 1, not just the zero value.
	got = New(testLookup, Config{Size: -3})
	require.Equal(t, 1, got.size)

	v, err := got.Lookup(t.Context(), "example.com")
	require.NoError(t, err)
	require.Equal(t, "example.com", v)
	require.Equal(t, 1, got.Len(), "a negative Size must behave as a capacity of 1")

	// A nil lookupFn is replaced by a default function that always fails.
	nilfn := New[string, any](nil, Config{Size: 1, TTL: 1 * time.Second})

	val, err := nilfn.Lookup(t.Context(), "example.com")
	require.ErrorIs(t, err, ErrNilLookupFunc)
	require.Nil(t, val)
}

func Test_Len(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, Config{Size: 3, TTL: 1 * time.Second})

	seed(c, map[string]*entry[any]{
		"example.com": {
			expireAt: time.Now(),
		},
		"example.net": {
			expireAt: time.Now(),
		},
	})
	require.Equal(t, 2, c.Len())
}

func Test_Reset(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, Config{Size: 1, TTL: 1 * time.Second})

	// One of EVERY kind of entry, so that Reset has all three queues and both maps to
	// clear. The revived entry is the one that matters: without it the stale queue is
	// already empty and clearing it is a no-op, so its removal would go unnoticed. A
	// queue left holding keys the map no longer has names them as victims, drops
	// nothing, and leaves the cache silently over capacity.
	seed(c, map[string]*entry[any]{
		"example.com": {
			expireAt: time.Now(),
		},
		"revived.example.com": {
			val:        "revived",
			staleUntil: time.Now().Add(time.Hour),
		},
		"residue.example.com": {
			err: errors.New("mock error"),
		},
	})
	fl := seedFlight(c, "inflight.example.com")

	c.Reset()

	require.Empty(t, c.keymap)
	require.Empty(t, c.flights)
	requireConsistentAccounting(t, c)

	// Reset must release the callers parked on the flights it invalidates: they
	// are waiting on a flight nobody will ever publish.
	requireFinished(t, fl, "Reset must finish the flights it invalidates")
}

func Test_Remove(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, Config{Size: 3, TTL: 1 * time.Second})

	seed(c, map[string]*entry[any]{
		"example.com": {
			expireAt: time.Now(),
		},
		"example.net": {
			expireAt: time.Now(),
		},
		"example.org": {
			expireAt: time.Now(),
		},
	})

	c.Remove("example.net")

	require.Equal(t, 2, c.Len())
	require.Contains(t, c.keymap, "example.com")
	require.Contains(t, c.keymap, "example.org")
	requireConsistentAccounting(t, c)
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

	c := New(lookupFn, Config{Size: 4, TTL: time.Minute})

	prodDone := make(chan struct{})

	go func() {
		defer close(prodDone)

		v, err := c.Lookup(context.Background(), "example.org")
		assert.NoError(t, err) // assert (not require) must be used off the test goroutine
		assert.Equal(t, "stale", v)
	}()

	// Wait until the producer has registered the in-flight entry.
	require.Eventually(t, func() bool {
		return inFlight(c, "example.org")
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

// Test_Reset_during_inflight_stale_flight_guard verifies that a flight
// invalidated by Reset cannot clobber the flight record of a successor
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

	c := New(lookupFn, Config{Size: 4, TTL: time.Minute})

	aDone := make(chan struct{})

	go func() {
		defer close(aDone)

		v, err := c.Lookup(context.Background(), "alpha")
		assert.NoError(t, err) // assert (not require) must be used off the test goroutine
		assert.Equal(t, "A", v, "the removed flight's result must still reach its caller")
	}()

	// Wait until flight A has registered its flight.
	require.Eventually(t, func() bool {
		return inFlight(c, "alpha")
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

	// Wait until flight B has registered itself.
	require.Eventually(t, func() bool {
		return inFlight(c, "alpha") && calls.Load() == 2
	}, time.Second, time.Millisecond)

	// Complete the stale flight A: it must not clobber B.
	close(relA)
	<-aDone

	require.True(t, inFlight(c, "alpha"), "a stale flight must not deregister its successor")

	close(relB)
	<-bDone

	v, err := c.Lookup(t.Context(), "alpha")
	require.NoError(t, err)
	require.Equal(t, "B", v, "the successor flight's value must be the cached one")
	require.Equal(t, int32(2), calls.Load())
}

func Test_PurgeExpired(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, Config{Size: 16, TTL: time.Minute})

	// SEVERAL of every expired kind, with distinct deadlines. With one of each, every
	// drain loop runs exactly once and can be degraded to an `if` — purging one entry
	// per kind, leaving the rest expired and returning a wrong count — unnoticed.
	fixture := map[string]*entry[any]{
		"fresh.example.com": {expireAt: time.Now().Add(30 * time.Second)},
	}

	for i := range 3 {
		age := time.Duration(i+1) * time.Second

		fixture[fmt.Sprintf("expired%d.example.com", i)] = &entry[any]{expireAt: time.Now().Add(-age)}
		fixture[fmt.Sprintf("stale%d.example.com", i)] = &entry[any]{
			val: "stale", staleUntil: time.Now().Add(30*time.Second + age),
		}
		fixture[fmt.Sprintf("residue%d.example.com", i)] = &entry[any]{err: errors.New("mock error")}
	}

	seed(c, fixture)

	fl := seedFlight(c, "inflight.example.com")

	require.Equal(t, 9, c.PurgeExpired(), "every expired, revived stale and error entry must be purged")

	require.Equal(t, 2, c.Len())
	require.Contains(t, c.keymap, "fresh.example.com")

	for i := range 3 {
		require.NotContains(t, c.keymap, fmt.Sprintf("expired%d.example.com", i))
		require.NotContains(t, c.keymap, fmt.Sprintf("stale%d.example.com", i))
		require.NotContains(t, c.keymap, fmt.Sprintf("residue%d.example.com", i))
	}

	require.True(t, inFlight(c, "inflight.example.com"), "in-flight lookups must not be purged")
	requireUnfinished(t, fl, "PurgeExpired must not disturb a lookup in flight")

	require.Equal(t, 0, c.PurgeExpired())
	requireConsistentAccounting(t, c)

	fl.finish()
}

// Test_Len_counts_flights_and_error_residue pins the documented Len
// semantics: in-flight lookups and expired error residue are counted.
func Test_Len_counts_flights_and_error_residue(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	lookupFn := func(_ context.Context, key string) (any, error) {
		if key == "slow.example.com" {
			<-release // hold the in-flight lookup open

			return key, nil
		}

		return nil, errors.New("mock error")
	}

	c := New(lookupFn, Config{Size: 4, TTL: time.Minute})

	done := make(chan struct{})

	go func() {
		defer close(done)

		_, _ = c.Lookup(context.Background(), "slow.example.com")
	}()

	// An in-flight lookup is counted by Len.
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

// Test_Reset_during_inflight_lookups pins that Reset does not leak the capacity
// accounting. Reset swaps in fresh maps, so the entries and flights it discards
// can never be dropped afterwards: the counters they fed must be cleared with
// the maps, or they describe a map that no longer exists and the capacity bound
// is broken for good.
func Test_Reset_during_inflight_lookups(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	lookupFn := func(_ context.Context, key string) (any, error) {
		if strings.HasPrefix(key, "slow") {
			<-release // hold the flight open across the Reset
		}

		return key, nil
	}

	c := New(lookupFn, Config{Size: 2, TTL: time.Minute})

	const flights = 3

	var wg sync.WaitGroup

	for i := range flights {
		wg.Go(func() {
			_, _ = c.Lookup(context.Background(), fmt.Sprintf("slow%d.example.com", i))
		})
	}

	require.Eventually(t, func() bool {
		return c.Len() == flights
	}, time.Second, time.Millisecond, "every flight must have registered its flight")

	c.Reset()

	requireConsistentAccounting(t, c)

	close(release)
	wg.Wait()

	requireConsistentAccounting(t, c)

	// The capacity bound must still hold. With a leaked residue index, the eviction
	// disagrees with the map and makeRoom stops doing its job.
	for i := range 3 * flights {
		_, err := c.Lookup(t.Context(), fmt.Sprintf("key%d.example.com", i))
		require.NoError(t, err)
	}

	require.LessOrEqual(t, c.Len(), 2, "Reset must not leak the in-flight count")
	requireConsistentAccounting(t, c)
}

// Test_Remove_releases_a_coalesced_waiter pins that invalidating a flight
// releases the callers ALREADY waiting on it, not merely the ones that arrive
// afterwards.
//
// Deregistering a flight is only observable by a caller that re-enters the wait
// loop, and a caller parked on the flight cannot re-enter until the flight's
// channel is closed. If only the producer could close it, an "invalidated" flight
// would still gate every waiter it had already collected — for the whole
// remaining life of a flight nobody will ever use, and forever if its lookup
// never returns, which is precisely the case Remove is documented to rescue.
func Test_Remove_releases_a_coalesced_waiter(t *testing.T) {
	t.Parallel()

	hang := make(chan struct{})
	defer close(hang)

	var calls atomic.Int32

	lookupFn := func(ctx context.Context, _ string) (string, error) {
		if calls.Add(1) == 1 {
			<-hang // the first lookup never returns

			return "", ctx.Err()
		}

		return "fresh", nil
	}

	c := New(lookupFn, Config{Size: 2, TTL: time.Minute})

	go func() {
		_, _ = c.Lookup(context.Background(), "key") // the producer, hanging forever
	}()

	require.Eventually(t, func() bool { return inFlight(c, "key") }, time.Second, time.Millisecond)

	waiter := make(chan string, 1)

	go func() {
		v, _ := c.Lookup(context.Background(), "key") // coalesces onto the hanging flight
		waiter <- v
	}()

	waitForParkedLookupWaiter(t, "Test_Remove_releases_a_coalesced_waiter")

	c.Remove("key")

	select {
	case v := <-waiter:
		require.Equal(t, "fresh", v, "the released waiter must run a fresh lookup")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "Remove did not release the caller already coalesced onto the flight it invalidated")
	}
}

// Test_Reset_releases_a_coalesced_waiter is the same property for Reset.
func Test_Reset_releases_a_coalesced_waiter(t *testing.T) {
	t.Parallel()

	hang := make(chan struct{})
	defer close(hang)

	var calls atomic.Int32

	lookupFn := func(ctx context.Context, _ string) (string, error) {
		if calls.Add(1) == 1 {
			<-hang

			return "", ctx.Err()
		}

		return "fresh", nil
	}

	c := New(lookupFn, Config{Size: 2, TTL: time.Minute})

	go func() {
		_, _ = c.Lookup(context.Background(), "key")
	}()

	require.Eventually(t, func() bool { return inFlight(c, "key") }, time.Second, time.Millisecond)

	waiter := make(chan string, 1)

	go func() {
		v, _ := c.Lookup(context.Background(), "key")
		waiter <- v
	}()

	waitForParkedLookupWaiter(t, "Test_Reset_releases_a_coalesced_waiter")

	c.Reset()

	select {
	case v := <-waiter:
		require.Equal(t, "fresh", v, "the released waiter must run a fresh lookup")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "Reset did not release the caller already coalesced onto the flight it invalidated")
	}
}

// Test_PurgeExpired_bulk pins the path a real bulk expiry takes: a cache filled in a
// burst with one TTL expires all at once, and sifting every entry out of the heap one
// at a time costs several times what rebuilding the heap around the survivors does —
// all of it under the exclusive write lock.
//
// The survivors must come out of that rebuild in deadline order, or the head of the
// queue is no longer the victim the whole eviction path takes it for.
func Test_PurgeExpired_bulk(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, Config{Size: 256, TTL: time.Minute})

	fixture := map[string]*entry[any]{}

	// Enough expired values to force the rebuild, and enough survivors — with
	// deadlines deliberately out of insertion order — to need re-sifting afterwards.
	for i := range 100 {
		fixture[fmt.Sprintf("expired%d.example.com", i)] = &entry[any]{
			expireAt: time.Now().Add(-time.Duration(i+1) * time.Second),
		}
	}

	for i := range 20 {
		fixture[fmt.Sprintf("valid%02d.example.com", i)] = &entry[any]{
			val:      i,
			expireAt: time.Now().Add(time.Duration((i*7)%20+1) * time.Minute),
		}
	}

	seed(c, fixture)

	require.Equal(t, 100, c.PurgeExpired(), "every expired value must go, not just the first few")
	require.Equal(t, 20, c.Len(), "and every valid one must stay")

	requireConsistentAccounting(t, c)

	// The rebuilt heap must still be a heap: its head is the survivor closest to
	// expiring, and it stays so as the survivors are drained in order.
	prev := time.Time{}

	for c.vic.values.len() > 0 {
		key, item, ok := c.vic.values.top()
		require.True(t, ok)
		require.Falsef(t, item.deadline().Before(prev),
			"the rebuilt queue is out of order at %v: its head is not the earliest deadline", key)

		prev = item.deadline()

		c.mux.Lock()
		c.drop(key)
		c.mux.Unlock()
	}
}

// Test_PurgeExpired_survives_a_drifted_queue pins the same guard makeRoom has: a queue
// naming a key the map does not hold must not be counted as purged, and must not spin.
func Test_PurgeExpired_survives_a_drifted_queue(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, Config{Size: 4, TTL: time.Minute})

	seed(c, map[string]*entry[any]{
		"expired.example.com": {expireAt: time.Now().Add(-time.Second)},
	})

	// Drift: the residue queue names an entry that was never stored.
	c.mux.Lock()
	c.vic.residue.push("ghost.example.com", &entry[any]{err: errors.New("ghost")})
	c.mux.Unlock()

	done := make(chan int, 1)

	go func() { done <- c.PurgeExpired() }()

	select {
	case purged := <-done:
		require.Equal(t, 1, purged, "the ghost must not be counted: nothing was dropped for it")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "PurgeExpired spun forever holding the write lock")
	}

	require.Zero(t, c.Len())
}

// Test_PurgeExpired_across_the_bulk_threshold sweeps the expiry ratio across the
// threshold at which PurgeExpired stops sifting entries out of the heap one at a time
// and starts rebuilding it around the survivors.
//
// The two paths are not independent: the sparse one pops heads until its budget runs
// out and then FALLS THROUGH to the partition, so at ratios near the threshold both
// run, and the partition has to be correct on a queue the pop loop has already
// modified.
//
// At every ratio the survivors must be exactly the unexpired entries, and what is
// left must still be a heap — because its head is what every eviction reads.
func Test_PurgeExpired_across_the_bulk_threshold(t *testing.T) {
	t.Parallel()

	const total = 4 * bulkPurgeRatio // enough that the budget lands inside the range

	for _, expired := range []int{
		0, 1, 2,
		(total / bulkPurgeRatio) - 1, // just under the budget: sifted out one by one
		total / bulkPurgeRatio,       // exactly the budget
		(total / bulkPurgeRatio) + 1, // just over: the pop loop falls through
		total / 4, total / 2, total - 1, total,
	} {
		c := New(nopLookupFn, Config{Size: 2 * total, TTL: time.Minute})

		fixture := make(map[string]*entry[any], total)
		valid := map[string]bool{}

		for i := range total {
			key := fmt.Sprintf("key%03d.example.com", i)

			if i < expired {
				// Distinct deadlines in the past, in an order unrelated to the key.
				fixture[key] = &entry[any]{expireAt: time.Now().Add(-time.Duration((i*7)%53+1) * time.Second)}
				continue
			}

			fixture[key] = &entry[any]{val: i, expireAt: time.Now().Add(time.Duration((i*11)%37+1) * time.Minute)}
			valid[key] = true
		}

		seed(c, fixture)

		require.Equalf(t, expired, c.PurgeExpired(), "expired=%d: wrong number purged", expired)
		require.Lenf(t, c.keymap, len(valid), "expired=%d: wrong number of survivors", expired)

		for key := range valid {
			require.Containsf(t, c.keymap, key, "expired=%d: %v was valid and must have survived", expired, key)
		}

		// Whatever path ran, what is left must still be a heap: the eviction policy
		// reads its head and takes it for the earliest deadline.
		requireQueueOrder(t, &c.vic.values)
		requireConsistentAccounting(t, c)
	}
}
