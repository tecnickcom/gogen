package sfcache

import "time"

// staleState carries the last known good value of the entry that a refresh flight
// is about to supersede, captured before it does, so that [Cache.publish] can
// revive it if the refresh fails.
type staleState[V any] struct {
	// until is the serving deadline already fixed when the flight started: the
	// value's original expiration plus [Config.MaxStale], or, for a value an
	// earlier failure already revived, the deadline fixed back then. The zero Time
	// means no deadline is fixed yet.
	until time.Time

	// val is the last known good value.
	val V

	// ok reports whether a stale value is available at all.
	ok bool

	// anchored reports whether until is final. A value revived by an earlier failure
	// keeps the window that failure gave it: were later failures allowed to re-anchor
	// it, a permanently failing upstream could serve it forever.
	anchored bool
}

// staleFrom captures the last known good value of the entry a refresh flight is
// about to supersede. A previously revived entry keeps the deadline anchored by
// the failure that revived it; a plain value carries its expiration-anchored
// deadline, if any, and stays open to a failure-anchored one (see
// [Cache.staleDeadline]). Error residue and missing entries yield no stale value.
// NOTE: this is not thread-safe, it should be called within a mutex lock.
func (c *Cache[K, V]) staleFrom(old *entry[V]) staleState[V] {
	// The stale-config test is redundant for the SERVING decision (with both windows
	// disabled staleDeadline can never revive) but not for RETENTION: it stops a cache
	// with no stale-if-error from copying V into the staleState and holding it for the
	// duration of every flight. Do not simplify it away.
	if ((c.maxStale <= 0) && (c.maxStaleOnFailure <= 0)) || (old == nil) || (old.err != nil) {
		return staleState[V]{}
	}

	if !old.staleUntil.IsZero() {
		return staleState[V]{val: old.val, until: old.staleUntil, ok: true, anchored: true}
	}

	stale := staleState[V]{val: old.val, ok: true}

	if c.maxStale > 0 {
		stale.until = old.expireAt.Add(c.maxStale)
	}

	return stale
}

// staleDeadline resolves the deadline until which the stale value carried by a
// failed flight may be served, and reports whether it may still be served now.
//
// A value no earlier failure has revived opens a failure-anchored window (see
// [Config.MaxStaleOnFailure]) measured from this failure; when an
// expiration-anchored window ([Config.MaxStale]) is also configured, the later of
// the two deadlines wins.
func (c *Cache[K, V]) staleDeadline(stale staleState[V]) (time.Time, bool) {
	if !stale.ok {
		return time.Time{}, false
	}

	now := time.Now()
	until := stale.until

	if !stale.anchored && (c.maxStaleOnFailure > 0) {
		if d := now.Add(c.maxStaleOnFailure); d.After(until) {
			until = d
		}
	}

	return until, now.Before(until)
}
