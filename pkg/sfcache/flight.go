package sfcache

import "sync"

// flight is a lookup in progress: the channel its producer closes when the lookup
// completes, and the guard that makes closing it idempotent.
//
// The producer always finishes its own flight, and [Cache.Remove] and [Cache.Reset]
// finish the flights they invalidate. They have to: a caller parked on a flight cannot
// notice that it was deregistered until the channel closes, so it would otherwise stay
// parked for the whole remaining life of a flight nobody will use.
type flight struct {
	// done is closed exactly once, by finish.
	done chan struct{}

	// once makes the close idempotent, so that a producer and an invalidator
	// racing to finish the same flight cannot double-close.
	once sync.Once
}

// newFlight creates a flight that no caller is waiting on yet.
func newFlight() *flight {
	return &flight{done: make(chan struct{})}
}

// finish releases every caller waiting on the flight. It is safe to call more than
// once, and from any goroutine.
//
// Whoever calls it must have deregistered the flight first, so that a woken caller
// re-reading the cache finds a terminal state instead of the flight it just left.
func (f *flight) finish() {
	f.once.Do(func() { close(f.done) })
}

// startFlight registers the in-flight lookup for the key and captures the last
// known good value it supersedes, so that a failed refresh can serve it.
// NOTE: it must be called with the write lock held; it releases it.
func (c *Cache[K, V]) startFlight(key K, fl *flight) staleState[V] {
	defer c.mux.Unlock()

	stale := c.staleFrom(c.keymap[key])

	// The flight supersedes the entry it refreshes, so a key is never in both maps
	// and a waiter can never be handed the value this flight is replacing.
	c.drop(key)

	c.flights[key] = fl

	return stale
}

// abortFlight deregisters the given in-flight lookup, leaving a terminal state
// (no entry, no flight) for its waiters to observe.
func (c *Cache[K, V]) abortFlight(key K, fl *flight) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if c.flights[key] == fl {
		// Only ever drop this flight's own record: the key may already have been
		// removed and taken over by a newer flight, which must survive this one's
		// unwind.
		delete(c.flights, key)
	}
}
