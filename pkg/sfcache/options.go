package sfcache

import "time"

// TTLFunc is the generic function signature for computing a per-entry TTL
// from the key and the value returned by a successful lookup.
type TTLFunc[K comparable, V any] func(key K, val V) time.Duration

// Option is a type to allow setting custom cache options.
type Option[K comparable, V any] func(c *Cache[K, V])

// WithTTLFunc overrides the cache-wide TTL for individual entries.
//
// After each successful lookup, ttlFn is called with the key and the value
// about to be cached: a positive result becomes that entry's TTL, while a
// zero or negative result falls back to the cache-wide TTL passed to [New].
// A nil ttlFn leaves the cache-wide TTL in effect for every entry.
// Values revived by [WithStaleIfError] bypass ttlFn: a stale serve is not a
// successful lookup.
//
// NOTE: ttlFn runs synchronously while the cache's internal lock is held:
// it must be fast (a slow ttlFn stalls all cache traffic) and it must not
// call any method of the same cache, which would self-deadlock.
//
// Use this when freshness is a property of the data itself, such as caching
// DNS records with their authoritative TTL or credentials with a known
// expiration.
func WithTTLFunc[K comparable, V any](ttlFn TTLFunc[K, V]) Option[K, V] {
	return func(c *Cache[K, V]) {
		c.ttlFn = ttlFn
	}
}

// WithStaleIfError enables serving the last known good value when a refresh
// fails: if a lookup for an expired key returns an error, the previous
// successful value is returned (with a nil error) instead, but only until
// its original expiration plus maxStale.
//
// The revived entry stays expired, so every subsequent call still attempts a
// fresh lookup (coalesced as usual): recovery is automatic on the first
// success, which resets the entry and its expiration. The stale window takes
// precedence over the context-induced waiter retry, so an upstream that
// hangs (every refresh failing by caller timeout) is still served stale;
// outside the window, context-induced failures keep their retry semantics.
// Entries whose last outcome was an error are never served stale. Callers
// cannot distinguish a stale value from a fresh one, and a coalesced waiter
// may observe the stale value marginally past maxStale (scheduling delay).
// A maxStale <= 0 disables the behavior (default).
//
// Stale protection is best-effort, not guaranteed retention: the retained
// value rides on expired entries and is therefore lost to capacity eviction
// (expired entries are evicted first), [Cache.PurgeExpired],
// [Cache.Remove], [Cache.Reset], and a panicking lookup function.
//
// NOTE: the type parameters cannot be inferred because maxStale does not
// mention them, so this option requires explicit instantiation matching the
// cache types, e.g.:
//
//	sfcache.New(lookupFn, 128, ttl, sfcache.WithStaleIfError[string, []string](time.Minute))
func WithStaleIfError[K comparable, V any](maxStale time.Duration) Option[K, V] {
	return func(c *Cache[K, V]) {
		c.maxStale = maxStale
	}
}
