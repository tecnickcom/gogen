package sfcache

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Lookup returns the value for the key, coalescing concurrent requests for the
// same key into a single external lookup.
//
// A non-expired cached value is returned immediately. Otherwise the caller either
// runs the lookup itself or waits for the one already in flight and takes its
// result. Successful values are cached for the TTL; errors are shared with the
// coalesced callers but never cached, so the next call retries.
//
// With stale-if-error enabled (see [Config.MaxStale] and
// [Config.MaxStaleOnFailure]), a failed refresh may instead return the last known
// good value with a NIL error.
//
// [ErrLookupAborted] is returned to a caller whose context ends while it WAITS for
// an in-flight lookup, or before its own lookup would start; the caller that RAN
// the lookup receives the lookup function's own error. See the package
// documentation for the full context semantics.
func (c *Cache[K, V]) Lookup(ctx context.Context, key K) (V, error) {
	//nolint:gocritic // dupSubExpr: the self-comparison is the point (see ErrInvalidKey).
	if key != key {
		// A key not equal to itself hashes to a slot nothing can find again: reject it
		// before it touches a map.
		var zero V

		return zero, ErrInvalidKey
	}

	// Fast path: a fresh cached value only requires the read lock. item.err is
	// necessarily nil here (an entry holding an error is stored already expired), but it
	// is returned rather than assumed.
	if item, ok := c.fresh(key); ok {
		return item.val, item.err
	}

	return c.lookupSlow(ctx, key)
}

// fresh returns the entry for the key if it holds a non-expired value.
//
// The read lock is released via defer because an unhashable key panics on any use, and
// the lock must not leak with the panic.
func (c *Cache[K, V]) fresh(key K) (*entry[V], bool) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	item, ok := c.keymap[key]

	return item, ok && item.usable(false)
}

// lookupSlow coalesces onto the lookup already in flight for the key, or performs
// the external lookup when the cache holds no usable entry.
func (c *Cache[K, V]) lookupSlow(ctx context.Context, key K) (V, error) {
	// Ask the context BEFORE taking the lock, and again after every wait, but never
	// while holding it: ctx.Err() runs code this package does not own, and code that
	// panics or blocks under the exclusive write lock wedges every other caller of the
	// cache, including a cache HIT taking the read lock.
	ctxErr := ctx.Err()

	c.mux.Lock()

	waited := false

	for {
		if item, ok := c.keymap[key]; ok && item.usable(waited) {
			c.mux.Unlock()
			return item.val, item.err
		}

		fl, inflight := c.flights[key]
		if !inflight {
			// Nobody is resolving the key: this caller performs the external lookup.
			return c.start(ctx, key, ctxErr)
		}

		c.mux.Unlock()

		err := c.await(ctx, fl)
		if err != nil {
			var zero V

			return zero, err
		}

		// Re-evaluate from scratch: the flight may have completed, been invalidated, or
		// been replaced by a new one. A finished flight is always deregistered first, so
		// this loop cannot find it again and spin on its closed channel.
		waited = true

		// The context may have ended while this caller waited, and it may be the one to
		// run the next lookup: ask again, still outside the lock.
		ctxErr = ctx.Err()

		c.mux.Lock()
	}
}

// start performs the external lookup as the key's single producer, unless the
// caller's context has already ended: no lookup is ever started with a dead context.
// NOTE: it must be called with the write lock held; it releases it.
func (c *Cache[K, V]) start(ctx context.Context, key K, ctxErr error) (V, error) {
	if ctxErr == nil {
		return c.fetch(ctx, key)
	}

	c.mux.Unlock()

	var zero V

	return zero, fmt.Errorf("%w: %w", ErrLookupAborted, ctxErr)
}

// await blocks until the flight completes, or until the caller's own context ends, in
// which case it returns [ErrLookupAborted]. A waiter never finishes the flight: it did
// not start it, and other callers may still be waiting on it.
//
// NOTE: when both channels are ready at once, select picks pseudo-randomly, so a waiter
// whose context ends as the flight completes may observe either outcome.
func (c *Cache[K, V]) await(ctx context.Context, fl *flight) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("%w: %w", ErrLookupAborted, ctx.Err())
	case <-fl.done:
		return nil
	}
}

// fetch performs the external lookup as the single producer for the key,
// registering the flight and then publishing its result.
// NOTE: it must be called with the write lock held; it releases it.
func (c *Cache[K, V]) fetch(ctx context.Context, key K) (V, error) {
	fl := newFlight()

	// finalized suppresses the abortFlight call once publish has finalized the flight.
	// It is an optimization: abortFlight only ever drops a flight that is still its own,
	// so calling it after publish would be a no-op, but it would take the write lock to
	// find that out on every lookup.
	finalized := false

	defer func() {
		if !finalized {
			// lookupFn or ttlFn panicked: deregister the flight so that waiters observe
			// a terminal state. The panic reaches the caller.
			//
			// NOTE: abortFlight re-locks the non-reentrant mutex, so no frame this defer
			// can unwind through may still hold the write lock. Both calls that can panic
			// — lookupFn and entryTTL — run before publish takes it.
			c.abortFlight(key, fl)
		}

		// Deregistration always precedes this, so a released caller re-reading the cache
		// cannot find the flight it left.
		fl.finish()
	}()

	stale := c.startFlight(key, fl)

	val, err := c.lookupFn(ctx, key)

	// Everything the caller supplies is run BEFORE the lock is taken, never under it:
	// Context.Err, the Unwrap and Is methods of the caller's error (which errors.Is
	// calls), and the TTL function. Code this package does not own, blocking under the
	// exclusive write lock, wedges every other caller of the cache.
	ctxInduced := (err != nil) && errors.Is(err, ctx.Err())

	// Only a successful lookup is cached, so only it needs a TTL — and only it may run
	// ttlFn, which must not see the value of a failed lookup. A panic here unwinds
	// through the defer above: the flight is deregistered, nothing is stored, and
	// nothing has been evicted on its behalf.
	var ttl time.Duration

	if err == nil {
		ttl = c.entryTTL(key, val)
	}

	val, err = c.publish(key, fl, val, err, stale, ctxInduced, ttl)

	finalized = true

	return val, err
}

// publish stores the outcome of the given flight and returns the outcome its
// caller must receive.
//
//   - A flight invalidated mid-flight (Remove or Reset) has its outcome returned to its
//     own caller but not cached.
//   - A failure with a stale value available revives it and returns it with a nil error,
//     whatever the failure's cause.
//   - A failure with the producing caller's own context error (ctxInduced) publishes
//     nothing, so that a coalesced waiter with a live context retries rather than
//     inherit a context-induced error.
//   - Every other outcome is stored and shared with the waiters.
//
// NOTE: ctxInduced and ttl are both computed by [Cache.fetch], OUTSIDE this lock,
// because computing either means running code this package does not own.
//
// ctxInduced is errors.Is against ctx.Err(), which matches sentinel identity rather than
// provenance, so an upstream error wrapping a context error is treated as
// context-induced when the producing context has also ended. Matching by identity alone
// would instead hand live waiters an error caused by ANOTHER caller's cancellation.
func (c *Cache[K, V]) publish(
	key K, fl *flight, val V, err error, stale staleState[V], ctxInduced bool, ttl time.Duration,
) (V, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if c.flights[key] != fl {
		return val, err
	}

	// This flight is over, whatever its outcome. Deregistering it here, under the
	// lock, is what lets fetch's defer finish it safely.
	delete(c.flights, key)

	if err == nil {
		c.set(key, val, nil, ttl)

		return val, nil
	}

	if until, revive := c.staleDeadline(stale); revive {
		// Revive the last known good value captured when the flight started. The revived
		// entry stays expired, so the next call attempts a fresh lookup, and it carries
		// the anchored deadline so that further failures cannot push it back.
		c.makeRoom(key, evictStale)

		c.store(key, &entry[V]{val: stale.val, staleUntil: until})

		return stale.val, nil
	}

	if ctxInduced {
		// Publish nothing: with no entry and no flight, a coalesced waiter with a
		// live context re-runs the lookup.
		return val, err
	}

	c.set(key, val, err, 0)

	return val, err
}
