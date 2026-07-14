package sfcache

import "time"

// TTLFunc is the generic function signature for computing a per-entry TTL
// from the key and the value returned by a successful lookup.
type TTLFunc[K comparable, V any] func(key K, val V) time.Duration

// Option is a type to allow setting custom cache options.
//
// Options carry the settings that depend on the cache's key and value types, so their
// type parameters are inferred from their own argument. Settings that mention neither K
// nor V cannot be inferred and live in [Config] instead.
type Option[K comparable, V any] func(c *Cache[K, V])

// WithTTLFunc overrides the cache-wide TTL for individual entries.
//
// After each successful lookup, ttlFn is called with the key and the value it
// returned: a positive result becomes that entry's TTL, while a zero or negative
// result falls back to the cache-wide [Config.TTL].
// A nil ttlFn leaves the cache-wide TTL in effect for every entry.
// Values revived by stale-if-error (see [Config.MaxStale] and
// [Config.MaxStaleOnFailure]) bypass ttlFn: a stale serve is not a successful
// lookup. So does a failed one: ttlFn never sees the value of a lookup that
// returned an error.
//
// It is called for every successful lookup, including one whose result is not
// ultimately cached because [Cache.Remove] or [Cache.Reset] invalidated it
// mid-flight.
//
// NOTE: ttlFn runs synchronously on the goroutine that performed the lookup, and NOT
// under the cache's lock, so it may call other methods of the same cache. It must not
// call [Cache.Lookup] for the key it is being called for, which self-deadlocks against
// that key's own flight.
// If it panics, the panic propagates to the caller that ran the lookup, nothing is
// stored (the value that lookup fetched is lost, and so is any stale-if-error protection
// the key had), and no entry is evicted on its behalf.
//
// Use this when freshness is a property of the data itself, such as caching
// DNS records with their authoritative TTL or credentials with a known
// expiration.
func WithTTLFunc[K comparable, V any](ttlFn TTLFunc[K, V]) Option[K, V] {
	return func(c *Cache[K, V]) {
		c.ttlFn = ttlFn
	}
}
