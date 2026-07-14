package sfcache

import "time"

// set stores the outcome of a completed lookup, making room for it first.
//
// The ttl is computed by the caller, OUTSIDE the lock (see [Cache.entryTTL]), and is
// ignored for a failed lookup.
// NOTE: this is not thread-safe, it should be called within a mutex lock.
func (c *Cache[K, V]) set(key K, val V, err error, ttl time.Duration) {
	// Only a successful lookup produces a cacheable value (a nil value is a value):
	// errors are never cached, so their entry is stored already expired.
	cacheable := err == nil
	level := evictWorthless

	if cacheable {
		level = evictValue
	}

	c.makeRoom(key, level)

	var expireAt time.Time

	if cacheable {
		// Anchor the deadline AFTER making room, so the cost of the eviction is not
		// charged to the value's own lifetime.
		expireAt = time.Now().Add(ttl)
	}

	c.store(key, &entry[V]{
		err:      err,
		expireAt: expireAt,
		val:      val,
	})
}

// entryTTL returns the time-to-live for a new entry: the per-entry override when a TTL
// function is configured and returns a positive duration (see [WithTTLFunc]), or the
// cache-wide default otherwise.
//
// A non-positive result is clamped to zero, so the value expires as it is stored rather
// than before it existed: an expireAt in the past would eat into the window
// [Config.MaxStale] anchors to it.
//
// NOTE: it runs the caller-supplied ttlFn, so it must NOT be called while holding the
// lock. ttlFn is code this package does not own: one that blocks on a lock the caller
// also takes around a call into this cache would wedge every other caller of the cache.
// ttlFn and c.ttl are fixed at construction, so reading them here is safe.
func (c *Cache[K, V]) entryTTL(key K, val V) time.Duration {
	if c.ttlFn != nil {
		if d := c.ttlFn(key, val); d > 0 {
			return d
		}
	}

	return max(0, c.ttl)
}

// store writes the entry for the key and files it in the queue its kind belongs to
// (see [victims.file]).
// NOTE: this is not thread-safe, it should be called within a mutex lock.
func (c *Cache[K, V]) store(key K, item *entry[V]) {
	c.drop(key) // take out the entry being replaced, if any

	c.keymap[key] = item

	c.vic.file(key, item)
}

// drop removes the entry for the key from the map and from the queue holding it, and
// reports whether there was one.
// NOTE: this is not thread-safe, it should be called within a mutex lock.
func (c *Cache[K, V]) drop(key K) bool {
	item, ok := c.keymap[key]
	if !ok {
		return false
	}

	delete(c.keymap, key)

	c.vic.unfile(item)

	return true
}

// makeRoom evicts entries until they fit the capacity target for storing the given key,
// taking nothing more valuable than the level allows. When the level permits no victim
// that exists, the cache is left over capacity.
//
// The loop cannot spin: every iteration either removes an entry from the map or returns,
// because it acts on what drop actually did rather than on what the queue named.
// NOTE: this is not thread-safe, it should be called within a mutex lock.
func (c *Cache[K, V]) makeRoom(key K, level evictLevel) {
	target := c.size

	if _, ok := c.keymap[key]; !ok {
		// this store adds an entry: make room for it
		target--
	}

	for len(c.keymap) > target {
		victim, ok := c.vic.pick(level, time.Now())
		if !ok {
			return
		}

		if !c.drop(victim) {
			return
		}
	}
}
