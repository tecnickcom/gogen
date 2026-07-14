package sfcache

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// payload is a value big enough that retaining one matters.
type payload struct{ _ [256]byte }

// collect runs the collector until the live count settles, and returns what is left.
func collect(live *atomic.Int64) int64 {
	remaining := live.Load()

	for range 20 {
		runtime.GC()

		remaining = live.Load()
		if remaining == 0 {
			return remaining
		}

		time.Sleep(10 * time.Millisecond)
	}

	return remaining
}

// Test_pqueue_retains_no_entry_it_gave_up pins the slot-clearing in [pqueue.remove] and
// [pqueue.partition].
//
// A queue that shrinks without zeroing the slots it vacates leaves the entries it gave
// up reachable through the slice's spare capacity, so they are never collected. Nothing
// observable says so: the map no longer holds them, [Cache.Len] returns zero, and
// [Cache.PurgeExpired] reports them all removed.
//
// The leak appears only when a queue SHRINKS, so removal is driven both ways: in bulk,
// through PurgeExpired's partition, and one at a time, through Remove.
func Test_pqueue_retains_no_entry_it_gave_up(t *testing.T) {
	t.Parallel()

	const size = 512

	tests := map[string]struct {
		ttl   time.Duration
		drain func(t *testing.T, c *Cache[int, *payload])
	}{
		// PurgeExpired takes the whole queue at once: drain, and the partition.
		"in bulk, through PurgeExpired": {
			ttl: 20 * time.Millisecond,
			drain: func(t *testing.T, c *Cache[int, *payload]) {
				t.Helper()

				time.Sleep(40 * time.Millisecond) // they all expire together

				require.Equal(t, size, c.PurgeExpired())
			},
		},
		// Remove takes them one at a time: the hole left in the middle of the heap is
		// filled with the last node, and the slot it vacated must be cleared.
		"one at a time, through Remove": {
			ttl: time.Hour,
			drain: func(t *testing.T, c *Cache[int, *payload]) {
				t.Helper()

				for i := range size {
					c.Remove(i)
				}
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var live atomic.Int64

			c := New(func(_ context.Context, _ int) (*payload, error) {
				val := &payload{}

				live.Add(1)

				runtime.AddCleanup(val, func(_ struct{}) { live.Add(-1) }, struct{}{})

				return val, nil
			}, Config{Size: size, TTL: tt.ttl})

			for i := range size {
				_, _ = c.Lookup(t.Context(), i)
			}

			require.EqualValues(t, size, live.Load(), "the fixture did not hold what it meant to")

			tt.drain(t, c)

			require.Zero(t, c.Len(), "the cache reports it holds nothing")

			remaining := collect(&live)

			// The CACHE must stay reachable across the collection above, or its queues die
			// with it and every value is collected whatever the queue cleared.
			runtime.KeepAlive(c)

			require.Zerof(t, remaining,
				"%d of %d values are still pinned through the queue's spare capacity, after the"+
					" cache reported every one of them gone", remaining, size)
		})
	}
}

// Test_entry_deadline pins what a queue orders an entry by. It is DERIVED from the
// outcome rather than stored beside it, which is what makes it impossible for a
// queued entry to carry a deadline that disagrees with the entry itself.
func Test_entry_deadline(t *testing.T) {
	t.Parallel()

	expireAt := time.Now().Add(time.Hour)
	staleUntil := time.Now().Add(time.Minute)

	value := &entry[string]{val: "v", expireAt: expireAt}
	require.False(t, value.revived())
	require.Equal(t, expireAt, value.deadline(), "an unrevived value is ordered by its expiration")

	// A revived value is stored already expired (a zero expireAt), so ordering it by
	// its expiration would put every one of them at the head of the queue in an
	// arbitrary order, and the victim among them would be whichever the heap happened
	// to hold first rather than the one whose window shuts first.
	revived := &entry[string]{val: "v", staleUntil: staleUntil}
	require.True(t, revived.revived())
	require.Equal(t, staleUntil, revived.deadline(), "a revived value is ordered by its anchored deadline")

	residue := &entry[string]{err: errors.New("mock error")}
	require.Equal(t, time.Time{}, residue.deadline(), "error residue carries no deadline: it is indexed, not queued")
}

// Test_pqueue_orders_by_deadline pins the one thing the whole eviction path rests
// on: the head of the queue is the entry with the earliest deadline. Everything else
// — which victim a store may take, and whether a store has a victim at all — is read
// off that head.
func Test_pqueue_orders_by_deadline(t *testing.T) {
	t.Parallel()

	now := time.Now()

	var q pqueue[string, string]

	// Pushed in an order that is neither sorted nor reversed.
	offsets := []int{5, 1, 9, 3, 7, 0, 8, 2, 6, 4}
	for _, off := range offsets {
		q.push(
			string(rune('a'+off)),
			&entry[string]{val: "v", expireAt: now.Add(time.Duration(off) * time.Minute)},
		)
	}

	require.Equal(t, len(offsets), q.len())

	// Draining the queue by its head must yield the deadlines in order.
	var got []string

	for q.len() > 0 {
		key, item, ok := q.top()
		require.True(t, ok)

		got = append(got, key)

		q.remove(item)
	}

	require.Equal(t, []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}, got,
		"the head of the queue must always be the earliest deadline")

	_, _, ok := q.top()
	require.False(t, ok, "an empty queue names no victim")
}

// Test_pqueue_remove_from_the_middle pins the property that makes the queues safe to
// hold entries at all: an entry that is replaced or removed is taken OUT of its
// queue, wherever it sits in it.
//
// The alternative — leaving it behind and checking on pop whether it is still the
// entry the map holds — is the lazy deletion this design exists to avoid: the queue
// then grows with the churn, and needs a compaction pass to stay bounded.
func Test_pqueue_remove_from_the_middle(t *testing.T) {
	t.Parallel()

	now := time.Now()

	var q pqueue[int, string]

	items := make([]*entry[string], 32)

	for i := range items {
		items[i] = &entry[string]{val: "v", expireAt: now.Add(time.Duration(i) * time.Minute)}
		q.push(i, items[i])
	}

	// Take out every third entry, from the middle of the heap rather than its head.
	removed := map[int]bool{}

	for i := 2; i < len(items); i += 3 {
		q.remove(items[i])

		removed[i] = true

		requireHeap(t, &q)
	}

	require.Equal(t, len(items)-len(removed), q.len(), "the queue must hold no superseded entry")

	// What is left must still drain in deadline order, and must be exactly what was
	// not removed.
	prev := -1

	for q.len() > 0 {
		key, item, ok := q.top()
		require.True(t, ok)
		require.Falsef(t, removed[key], "%d was removed but the queue still named it", key)
		require.Greater(t, key, prev, "the queue is out of order after removals from the middle")

		prev = key

		q.remove(item)
	}
}

// Test_pqueue_remove_the_last_entry covers the path where the entry removed IS the
// last node, so nothing has to be moved into the hole it leaves.
func Test_pqueue_remove_the_last_entry(t *testing.T) {
	t.Parallel()

	now := time.Now()

	var q pqueue[string, string]

	first := &entry[string]{val: "first", expireAt: now.Add(time.Minute)}
	second := &entry[string]{val: "second", expireAt: now.Add(time.Hour)}

	q.push("first", first)
	q.push("second", second)

	q.remove(second) // the tail

	require.Equal(t, 1, q.len())

	key, item, ok := q.top()
	require.True(t, ok)
	require.Equal(t, "first", key)
	require.Same(t, first, item)

	q.remove(first) // the head, which is also now the tail

	require.Zero(t, q.len())
}

// requireHeap checks the heap property and that every node records its own position,
// which is what makes remove exact.
func requireHeap[K comparable, V any](t *testing.T, q *pqueue[K, V]) {
	t.Helper()

	for idx, node := range q.nodes {
		require.Equal(t, idx, node.item.idx, "a queued entry lost track of its position")

		if idx == 0 {
			continue
		}

		parent := (idx - 1) / 2
		require.False(t, node.item.deadline().Before(q.nodes[parent].item.deadline()),
			"the queue is out of order: its head is not the earliest deadline")
	}
}

// Test_staleVictim pins the choice between the two kinds of value that are expired
// but can still be served: the ones no failure has revived, and the ones it has.
// Each of the three shapes the cache can be in is a different branch.
func Test_staleVictim(t *testing.T) {
	t.Parallel()

	now := time.Now()

	victim := func(t *testing.T, c *Cache[string, any]) (string, bool) {
		t.Helper()

		c.mux.Lock()
		defer c.mux.Unlock()

		return c.vic.staleVictim(time.Now())
	}

	t.Run("only a revived value", func(t *testing.T) {
		t.Parallel()

		// The values queue holds nothing expired, so the only value that can be given
		// up is one a failure revived.
		c := New(nopLookupFn, Config{Size: 4, MaxStaleOnFailure: time.Hour})
		seed(c, map[string]*entry[any]{
			"valid.example.com":   {val: "valid", expireAt: now.Add(time.Hour)},
			"revived.example.com": {val: "revived", staleUntil: now.Add(time.Minute)},
		})

		key, ok := victim(t, c)
		require.True(t, ok)
		require.Equal(t, "revived.example.com", key)
	})

	t.Run("only an unrevived value", func(t *testing.T) {
		t.Parallel()

		c := New(nopLookupFn, Config{Size: 4, MaxStaleOnFailure: time.Hour})
		seed(c, map[string]*entry[any]{
			"valid.example.com":   {val: "valid", expireAt: now.Add(time.Hour)},
			"expired.example.com": {val: "expired", expireAt: now.Add(-time.Second)},
		})

		key, ok := victim(t, c)
		require.True(t, ok)
		require.Equal(t, "expired.example.com", key)
	})

	t.Run("nothing expired at all", func(t *testing.T) {
		t.Parallel()

		c := New(nopLookupFn, Config{Size: 4, MaxStaleOnFailure: time.Hour})
		seed(c, map[string]*entry[any]{
			"valid.example.com": {val: "valid", expireAt: now.Add(time.Hour)},
		})

		_, ok := victim(t, c)
		require.False(t, ok, "a cache with nothing expired offers no stale victim")
	})

	t.Run("the revived value is kept when a caller has asked for it", func(t *testing.T) {
		t.Parallel()

		// Both kinds are present. The revived one is the only one carrying evidence
		// that anybody wants it, so the unrevived one goes first.
		c := New(nopLookupFn, Config{Size: 4, MaxStaleOnFailure: time.Hour})
		seed(c, map[string]*entry[any]{
			"unasked.example.com": {val: "unasked", expireAt: now.Add(-time.Second)},
			"revived.example.com": {val: "revived", staleUntil: now.Add(time.Minute)},
		})

		key, ok := victim(t, c)
		require.True(t, ok)
		require.Equal(t, "unasked.example.com", key,
			"the value nobody has asked for must go before the one being served")
	})

	t.Run("under MaxStale the unrevived value still goes first", func(t *testing.T) {
		t.Parallel()

		// Under MaxStale BOTH candidates carry a deadline — each one its own
		// expiration plus the same window — so comparing them degenerates into
		// "whichever was fetched first", which says nothing about worth.
		//
		// The demand signal must win anyway: the revived value is the one a caller
		// asked for during the outage, and evicting it costs that caller an error
		// immediately, while the unrevived one serves nobody. Ranking these two by
		// remaining window is ranking them by how little anyone wants them.
		//
		c := New(nopLookupFn, Config{Size: 4, MaxStale: time.Hour})
		seed(c, map[string]*entry[any]{
			// unrevived: expired an hour ago, so its window shuts LATER than the
			// revived one's. Nobody has asked for it since it expired.
			"unasked.example.com": {val: "unasked", expireAt: now.Add(-time.Second)},
			// revived: a caller asked for it during the outage, and its window shuts
			// sooner. It must survive anyway.
			"served.example.com": {val: "served", staleUntil: now.Add(time.Minute)},
		})

		key, ok := victim(t, c)
		require.True(t, ok)
		require.Equal(t, "unasked.example.com", key,
			"the value nobody has asked for must go first, whatever window it has left")
	})
}

// Test_pqueue_remove_sifts_the_replacement_towards_the_head pins the half of
// [pqueue.remove] that nothing else reaches: the node moved into the hole may belong
// ABOVE it, not below.
//
// Every other fixture removes entries in ascending order, so the replacement always
// sifts down and the up-sift, though covered, is never observed. A queue that has lost
// its heap property still drains in the right order often enough to fool a drain-order
// assertion, so this asserts the invariant itself.
func Test_pqueue_remove_sifts_the_replacement_towards_the_head(t *testing.T) {
	t.Parallel()

	now := time.Now()

	var q pqueue[int, string]

	items := map[int]*entry[string]{}

	// Pushed in this order the array is exactly [1, 10, 2, 11, 12, 3, 4].
	for _, minutes := range []int{1, 10, 2, 11, 12, 3, 4} {
		items[minutes] = &entry[string]{val: "v", expireAt: now.Add(time.Duration(minutes) * time.Minute)}
		q.push(minutes, items[minutes])
	}

	requireHeap(t, &q)
	require.Equal(t, 4, items[12].idx, "the fixture must place 12 where its removal pulls the tail up")

	// Removing index 4 fills the hole with the tail (4), whose parent is 10: it must
	// sift UP. down() alone cannot move it — index 4 is a leaf.
	q.remove(items[12])

	requireHeap(t, &q)

	key, _, ok := q.top()
	require.True(t, ok)
	require.Equal(t, 1, key, "the head must still be the earliest deadline")
}

// Test_pqueue_head_is_the_minimum_under_churn is the property the whole eviction path
// reads off: whatever has been pushed and removed, the head is the earliest deadline.
// A heap that has silently lost its invariant evicts the wrong victim AND reports
// "nothing is worthless" while holding worthless entries.
func Test_pqueue_head_is_the_minimum_under_churn(t *testing.T) {
	t.Parallel()

	now := time.Now()

	var q pqueue[int, string]

	live := map[int]*entry[string]{}

	// A deterministic but order-hostile sequence: deadlines unrelated to push order.
	for round := range 300 {
		key := (round * 37) % 101

		if item, held := live[key]; held {
			q.remove(item)
			delete(live, key)
		} else {
			item := &entry[string]{val: "v", expireAt: now.Add(time.Duration((key*7)%53) * time.Minute)}
			live[key] = item
			q.push(key, item)
		}

		requireHeap(t, &q)

		head, item, ok := q.top()
		if !ok {
			continue
		}

		for other, candidate := range live {
			require.Falsef(t, candidate.deadline().Before(item.deadline()),
				"the head (%d) is not the earliest deadline: %d is earlier", head, other)
		}
	}
}
