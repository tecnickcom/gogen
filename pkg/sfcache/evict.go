package sfcache

import "time"

// evictLevel is how much a store may evict to make room for itself: the more valuable
// the entry it adds, the more valuable the entry it may take.
type evictLevel uint8

const (
	// evictWorthless may take only entries that hold nothing worth keeping (see
	// [victims.worthlessAt]). It is what the residue of a failed lookup gets. When
	// nothing is worthless, the cache is left over capacity instead.
	evictWorthless evictLevel = iota

	// evictStale may also take a value that is only being served stale, but never a
	// valid one. It is what a stale revive gets.
	evictStale

	// evictValue may also take the valid entry closest to expiring. It is what a
	// successful lookup gets.
	evictValue
)

// victims holds every stored entry, each in exactly one of three queues, and chooses the
// one a store may take. The queues ARE the eviction policy: a victim is always the HEAD
// of one of them, so it is never searched for.
//
// It holds no reference to the cache's keymap, so an eviction cannot walk the cache to
// find its victim. Note that the queues themselves hold every stored entry, so a range
// over one of them is the same O(Size) pass under the same exclusive write lock.
// NOTE: this is not thread-safe, it must be used within the cache's mutex lock.
type victims[K comparable, V any] struct {
	// values holds every stored value no failed refresh has revived, ordered by
	// expiration.
	//
	// INVARIANT: every value in here is unrevived, so [victims.worthlessAt] takes the
	// same branch for all of them and is therefore monotonic in the expiration they are
	// sorted by. The head is at once the valid entry closest to expiring, the first to
	// turn worthless, and the least valuable unrevived stale value. This is what lets
	// every eviction question be answered by a queue head.
	values pqueue[K, V]

	// stale holds the values a failed refresh revived, ordered by the deadline that
	// failure anchored on them. Its head is the one whose window shuts first.
	//
	// They are kept apart from values because their deadline is anchored by the failure,
	// not by their expiration, so the two orders are unrelated.
	stale pqueue[K, V]

	// residue holds the entries that hold an error. Such an entry is worthless from the
	// moment it is stored, under every configuration, so the order here is degenerate
	// and the head is any of them.
	//
	// It is a queue rather than a map because Go never shrinks a map's table: finding
	// the one key left in a map an outage grew to Size would cost O(Size) on every
	// failing lookup thereafter.
	residue pqueue[K, V]

	// maxStale and maxStaleOnFailure are the cache's stale-if-error settings, fixed at
	// construction (see [Config.MaxStale] and [Config.MaxStaleOnFailure]).
	maxStale          time.Duration
	maxStaleOnFailure time.Duration
}

// reset empties every queue, keeping the settings.
func (v *victims[K, V]) reset() {
	v.values.reset()
	v.stale.reset()
	v.residue.reset()
}

// file puts the entry in the queue its kind belongs to. What an entry holds decides
// where it goes, so the three queues partition the stored entries.
//
// INVARIANT: this must stay in step with [victims.unfile], which re-derives the queue
// from the same fields. They hold because err and staleUntil are set at construction and
// never written again.
func (v *victims[K, V]) file(key K, item *entry[V]) {
	switch {
	case item.err != nil:
		v.residue.push(key, item)
	case item.revived():
		v.stale.push(key, item)
	default:
		v.values.push(key, item)
	}
}

// unfile takes the entry out of the queue holding it, by the same test that filed it.
func (v *victims[K, V]) unfile(item *entry[V]) {
	switch {
	case item.err != nil:
		v.residue.remove(item)
	case item.revived():
		v.stale.remove(item)
	default:
		v.values.remove(item)
	}
}

// worthlessAt returns the time from which the entry stops being worth keeping, and
// whether such a time exists at all. It is a property of the entry AND of the cache's
// stale-if-error settings: a value whose TTL ran out is still worth keeping while a
// stale window can serve it.
//
// The case ORDER is load-bearing: with BOTH windows configured, a value whose MaxStale
// window has closed is still not worthless, because the next failed refresh opens a
// MaxStaleOnFailure window on it.
func (v *victims[K, V]) worthlessAt(item *entry[V]) (time.Time, bool) {
	switch {
	case item.err != nil:
		// Error residue is never served stale: worth nothing from the moment it is
		// stored.
		return time.Time{}, true
	case item.revived():
		// Already revived: worth keeping until the deadline that failure anchored.
		return item.staleUntil, true
	case v.maxStaleOnFailure > 0:
		// The next failed refresh, whenever it comes, can still revive this value:
		// it never becomes worthless on its own.
		return time.Time{}, false
	case v.maxStale > 0:
		// Worth keeping until its expiration-anchored stale deadline.
		return item.expireAt.Add(v.maxStale), true
	default:
		// No stale-if-error: worth nothing once it expires.
		return item.expireAt, true
	}
}

// worthless reports whether the entry holds nothing worth keeping any more.
func (v *victims[K, V]) worthless(item *entry[V], now time.Time) bool {
	at, ok := v.worthlessAt(item)

	return ok && !now.Before(at)
}

// pick names the least valuable entry the level may take, and reports whether there is
// one: a worthless entry, otherwise a value that is merely being served stale, and
// finally the valid entry closest to expiring.
//
// Every candidate is the head of a queue, so an eviction costs O(log n) and a store with
// no victim it may take says so in constant time. A lookup in flight is never named: it
// holds no entry.
func (v *victims[K, V]) pick(level evictLevel, now time.Time) (K, bool) {
	if key, ok := v.worthlessVictim(now); ok {
		return key, true
	}

	if level < evictStale {
		var zero K

		return zero, false
	}

	if key, ok := v.staleVictim(now); ok {
		return key, true
	}

	if level < evictValue {
		var zero K

		return zero, false
	}

	// Nothing is worthless and nothing is expired-but-servable, so every value is
	// valid, and the head of the queue is the one closest to expiring: the victim
	// whose loss costs the fewest hits.
	if key, _, ok := v.values.top(); ok {
		return key, true
	}

	var zero K

	return zero, false
}

// worthlessVictim names an entry nobody can ever be served: the only eviction a failed
// lookup may make.
func (v *victims[K, V]) worthlessVictim(now time.Time) (K, bool) {
	// Error residue is worthless under every configuration, from the moment it is
	// stored, so any of it will do.
	if key, _, ok := v.residue.top(); ok {
		return key, true
	}

	// A revived value whose window has shut. The queue is ordered by that deadline, so
	// if the head has not shut, none has.
	if key, item, ok := v.stale.top(); ok && v.worthless(item, now) {
		return key, true
	}

	// A value whose stale window has shut, or that never had one. The queue's
	// monotonicity means that if the head has not turned worthless, none has.
	if key, item, ok := v.values.top(); ok && v.worthless(item, now) {
		return key, true
	}

	var zero K

	return zero, false
}

// staleVictim names a value that is expired but can still be served stale. It is called
// only once nothing worthless is left, so both candidates it weighs are still servable.
//
// The value NO FAILURE HAS REVIVED goes first, always. A value is revived because a
// caller asked for it during the outage, so a revived value is the only one carrying
// evidence that anybody wants it. Evicting the unasked-for value costs nothing until
// someone asks; evicting a revived one costs its next caller an error immediately.
func (v *victims[K, V]) staleVictim(now time.Time) (K, bool) {
	if key, item, ok := v.values.top(); ok && item.expired(now) {
		return key, true
	}

	// Only revived values are left to give up: take the one whose window shuts first.
	if key, _, ok := v.stale.top(); ok {
		return key, true
	}

	var zero K

	return zero, false
}
