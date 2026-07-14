// Shared fixtures and invariant checks for the sfcache tests.

package sfcache

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// nopLookupFn is a placeholder lookup function for tests that never invoke it.
func nopLookupFn(_ context.Context, _ string) (any, error) {
	return nil, nil //nolint:nilnil
}

// seed replaces the cache contents with the given completed entries.
// Fixtures must go through it rather than assigning keymap directly: the queues and
// the residue index are only kept in step by store and drop, and a direct
// assignment leaves an entry filed nowhere.
func seed[K comparable, V any](c *Cache[K, V], entries map[K]*entry[V]) {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.keymap = make(map[K]*entry[V], c.size)
	c.flights = make(map[K]*flight)

	c.vic.reset()

	for key, item := range entries {
		c.store(key, item)
	}
}

// seedFlight registers an in-flight lookup for the key, exactly as startFlight
// does: the flight supersedes the entry it refreshes, so that a key is never in
// both maps. The caller plays the producer and must finish the returned flight,
// which is idempotent and therefore safe even if Remove or Reset got there
// first.
func seedFlight[K comparable, V any](c *Cache[K, V], key K) *flight {
	c.mux.Lock()
	defer c.mux.Unlock()

	fl := newFlight()

	c.drop(key)

	c.flights[key] = fl

	return fl
}

// requireFinished checks that the flight has been finished, i.e. that every
// caller parked on it has been released.
func requireFinished(t *testing.T, fl *flight, msg string) {
	t.Helper()

	select {
	case <-fl.done:
	default:
		require.Fail(t, msg)
	}
}

// requireUnfinished checks that the flight is still holding its waiters.
func requireUnfinished(t *testing.T, fl *flight, msg string) {
	t.Helper()

	select {
	case <-fl.done:
		require.Fail(t, msg)
	default:
	}
}

// inFlight reports whether a lookup for the key is currently registered.
func inFlight[K comparable, V any](c *Cache[K, V], key K) bool {
	c.mux.RLock()
	defer c.mux.RUnlock()

	_, ok := c.flights[key]

	return ok
}

// requireConsistentAccounting checks the invariant the whole eviction path rests
// on: every stored entry is filed in exactly one of the two queues or the residue
// index, according to what it holds; each queued entry sits where it says it does;
// and each queue is ordered, so that its head really is the victim it is taken for.
//
// It is the backstop against a mutation of keymap that bypasses store/drop, so it
// must be called after anything that can change the map — including [Cache.Reset],
// [Cache.Remove], and [Cache.PurgeExpired].
func requireConsistentAccounting[K comparable, V any](t *testing.T, c *Cache[K, V]) {
	t.Helper()

	c.mux.RLock()
	defer c.mux.RUnlock()

	residue, values, stale := 0, 0, 0

	for key, item := range c.keymap {
		switch {
		case item.err != nil:
			residue++

			requireQueued(t, &c.vic.residue, key, item)
		case item.revived():
			stale++

			requireQueued(t, &c.vic.stale, key, item)
		default:
			values++

			requireQueued(t, &c.vic.values, key, item)
		}

		_, inflight := c.flights[key]
		require.Falsef(t, inflight, "%v is both stored and in flight: the two maps must be disjoint", key)
	}

	require.Equal(t, residue, c.vic.residue.len(), "the residue queue must match the map")
	require.Equal(t, values, c.vic.values.len(), "the values queue must match the map")
	require.Equal(t, stale, c.vic.stale.len(), "the stale queue must match the map")

	requireQueueOrder(t, &c.vic.values)
	requireQueueOrder(t, &c.vic.stale)
	requireQueueOrder(t, &c.vic.residue)

	// The three queues are a PARTITION: an entry that holds an error is worthless
	// under every configuration and must never be ordered against a value, whose
	// deadline means something. Nothing may cross.
	for _, node := range c.vic.values.nodes {
		require.NoErrorf(t, node.item.err, "%v holds an error but is queued as a value", node.key)
		require.Falsef(t, node.item.revived(), "%v was revived but is queued as an unrevived value", node.key)
	}

	for _, node := range c.vic.stale.nodes {
		require.NoErrorf(t, node.item.err, "%v holds an error but is queued as a stale value", node.key)
		require.Truef(t, node.item.revived(), "%v is queued as revived but was not", node.key)
	}

	for _, node := range c.vic.residue.nodes {
		require.Errorf(t, node.item.err, "%v is queued as residue but holds no error", node.key)
	}
}

// evictOne names the least valuable entry the level may take and drops it, reporting
// whether one was taken. It is what [Cache.makeRoom] does per iteration, spelled out
// so that the tests can drive a single eviction.
func evictOne[K comparable, V any](c *Cache[K, V], level evictLevel) bool {
	key, ok := c.vic.pick(level, time.Now())
	if !ok {
		return false
	}

	return c.drop(key)
}

// requireQueued checks that the entry sits in the queue at the position it records,
// which is what makes its removal exact: an entry that lost track of its position
// would be left behind in the queue when it is dropped, and the queue would then
// hand a dropped key to the next eviction.
func requireQueued[K comparable, V any](t *testing.T, q *pqueue[K, V], key K, item *entry[V]) {
	t.Helper()

	require.GreaterOrEqual(t, item.idx, 0)
	require.Lessf(t, item.idx, q.len(), "%v records a position past the end of its queue", key)
	require.Samef(t, item, q.nodes[item.idx].item, "%v is not at the position it records", key)
	require.Equal(t, key, q.nodes[item.idx].key)
}

// requireQueueOrder checks the heap property — the head is the earliest deadline —
// and that every node records its own position.
func requireQueueOrder[K comparable, V any](t *testing.T, q *pqueue[K, V]) {
	t.Helper()

	for idx, node := range q.nodes {
		require.Equal(t, idx, node.item.idx, "a queued entry lost track of its position")

		if idx == 0 {
			continue
		}

		parent := (idx - 1) / 2
		require.Falsef(t, node.item.deadline().Before(q.nodes[parent].item.deadline()),
			"the queue is out of order at %d: its head is not the earliest deadline", idx)
	}
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

// values counts the entries that hold a value the cache could still serve,
// i.e. everything but the residue of a failed lookup. This is what Config.Size
// bounds, give or take the one value a stale revive may add (see the package doc).
func values[K comparable, V any](c *Cache[K, V]) int {
	c.mux.RLock()
	defer c.mux.RUnlock()

	return len(c.keymap) - c.vic.residue.len()
}

// anyInFlight reports whether any lookup is still in flight.
func anyInFlight[K comparable, V any](c *Cache[K, V]) bool {
	c.mux.RLock()
	defer c.mux.RUnlock()

	return len(c.flights) > 0
}
