package sfcache

import "time"

// entry is the completed outcome of a lookup for a single key: a value, or the residue
// of a failure.
//
// INVARIANT: the OUTCOME is immutable once stored. An update replaces the whole entry,
// so an *entry read under the lock stays valid after the lock is released, and the
// queues can classify an entry by its fields without fear of it changing underneath
// them. idx is not part of the outcome: it records where the entry is HELD, is written
// only under the write lock, and is never read outside it.
type entry[V any] struct {
	// err is the error returned by the external lookup.
	err error

	// expireAt is the expiration deadline (monotonic clock). The zero Time marks
	// an entry stored already expired: error residue, and a revived stale value.
	expireAt time.Time

	// staleUntil is set only on a value revived by a failed refresh: the deadline
	// until which further failed refreshes may keep serving it. The zero Time
	// means the entry was not revived, so its stale window is not anchored yet.
	staleUntil time.Time

	// val is the value associated with the key.
	val V

	// idx is the entry's position in the [pqueue] holding it, which makes removing it
	// from the middle of a queue O(log n).
	idx int
}

// usable reports whether the caller can return this entry: a non-expired value,
// or, for a caller that awaited the flight that produced it, any completed
// outcome, even an expired one (a TTL <= 0, or an error that is not cached).
func (e *entry[V]) usable(waited bool) bool {
	return waited || time.Now().Before(e.expireAt)
}

// expired reports whether the entry can no longer be served fresh: what usable
// tests, negated, at an explicit time.
func (e *entry[V]) expired(now time.Time) bool {
	return !now.Before(e.expireAt)
}

// revived reports whether a failed refresh brought this value back to be served
// stale, which is what anchored its deadline and what moved it to the stale queue.
func (e *entry[V]) revived() bool {
	return !e.staleUntil.IsZero()
}

// deadline is what the queue holding this entry orders it by: the deadline a failure
// anchored on a revived value, and the expiration of every other.
//
// It is DERIVED from the outcome rather than stored beside it, so a queued entry cannot
// go out of order and no second copy of it can drift.
func (e *entry[V]) deadline() time.Time {
	if e.revived() {
		return e.staleUntil
	}

	return e.expireAt
}
