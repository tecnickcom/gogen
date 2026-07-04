/*
Package sfcache provides a local, thread-safe, fixed-size cache for expensive
lookups with single-flight deduplication.

# Problem

Services that repeatedly fetch the same external value (DNS, secrets, remote
metadata, API responses) often suffer from duplicated work under concurrency.
Without coordination, multiple goroutines can trigger identical slow lookups at
the same time, increasing latency, cost, and upstream load.

sfcache solves this by combining TTL caching, bounded memory, and in-flight
request coalescing for identical keys.

# How It Works

Create a cache with [New], providing:

  - a [LookupFunc] that performs the external lookup,
  - a maximum entry count (`size`),
  - a time-to-live (`ttl`) for successful values,
  - optional behavior overrides such as [WithTTLFunc] (per-entry TTLs) and
    [WithStaleIfError] (serve the last known good value on refresh failure).

On [Cache.Lookup]:

 1. If a non-expired entry exists, the cached value is returned immediately.
 2. If the key is being resolved by another goroutine, duplicate callers wait
    and receive that same result (single-flight behavior).
 3. On miss or expiry, one lookup function call is executed and its result is
    stored in cache.
 4. If cache capacity is reached, eviction removes an expired entry first, or
    otherwise the oldest entry by expiration deadline.

# Key Features

  - Fixed-size local cache with explicit capacity to avoid unbounded memory
    growth.
  - Internal synchronization for safe concurrent access without external locks.
  - Lock-friendly fast path: cache hits only acquire a read lock.
  - Single-flight request collapsing for duplicate in-flight lookups.
  - TTL-based freshness with automatic refresh on next miss after expiry,
    with optional per-entry TTLs via [WithTTLFunc].
  - Optional stale-if-error resilience via [WithStaleIfError].
  - Monotonic-clock expiration, immune to wall-clock adjustments (e.g. NTP).
  - Explicit cache control via [Cache.Remove], [Cache.Reset], and
    [Cache.PurgeExpired].

# Semantics and Caveats

  - Only successful values are cached for the TTL (including legitimate nil
    values). Lookup errors are shared with all coalesced callers but never
    cached (no negative caching); a failed key leaves an already-expired entry
    behind until it is lazily evicted or overwritten, and at capacity this
    residue can displace a healthy entry.
  - Whatever the lookup function returns is passed through as-is, including a
    non-nil value returned alongside a non-nil error.
  - A ttl <= 0 disables value caching: every call triggers a new lookup, but
    concurrent callers for the same key are still coalesced. Two options
    qualify this: [WithTTLFunc] can still assign individual entries a
    positive TTL, and with [WithStaleIfError] the previous value is retained
    and can be served after a failed refresh. A caller that waited for an
    in-flight lookup accepts the latest completed outcome for the key, which
    under heavy churn may come from a later flight than the one it first
    awaited.
  - The external lookup runs under the context of the caller that started it.
    If the lookup fails with that context's error while coalesced waiters are
    queued, the error is not shared: one of the waiters retries the lookup
    with its own context. A caller whose own context ends receives
    [ErrLookupAborted]; no external lookup is ever started with an
    already-ended context, while fresh cached values are served regardless of
    context state.
  - If a waiting caller's context ends at the same instant the awaited
    lookup completes, either outcome may be observed: the caller can receive
    [ErrLookupAborted] even though a result just became available, or the
    result even though its context just ended (select nondeterminism).
  - Expiration uses the monotonic clock, which on most platforms (e.g. Linux
    CLOCK_MONOTONIC) does not advance while the system is suspended: TTLs
    are effectively extended by the time spent in suspend. This matters on
    laptops and sleeping virtual machines, not on always-on servers.
  - The lookup function must honor context cancellation and eventually
    return: a lookup that hangs forever pins its key (in-flight entries never
    expire and are never evicted) until [Cache.Remove] or [Cache.Reset]. It
    must not call [Cache.Lookup] for the same key of the same cache, which
    would self-deadlock. If it panics, the panic propagates to the caller
    that ran it and waiters retry.
  - With [WithStaleIfError], a failed refresh serves the last known good
    value (with a nil error) until its original expiration plus maxStale;
    the revived entry stays expired so every call still attempts a refresh,
    and the first success replaces it. The stale window takes precedence
    over the context-induced retry, so an upstream that hangs (every
    refresh dying by caller timeout) is still served stale. Error residue
    is never served stale, and callers cannot distinguish a stale value
    from a fresh one. Stale protection is best-effort: the value is lost to
    capacity eviction (expired entries are evicted first),
    [Cache.PurgeExpired], [Cache.Remove], [Cache.Reset], and a panicking
    lookup function.
  - [Cache.Remove] and [Cache.Reset] also invalidate in-flight lookups: the
    result of a removed flight is returned to the caller that performed it
    but not cached, and coalesced waiters retry with a fresh lookup.
  - The capacity bound can be exceeded while more than `size` distinct keys
    are being resolved at once, as in-flight entries are never evicted; the
    excess is reclaimed as those lookups complete and new entries are stored.
    Expired entries are removed lazily (or via [Cache.PurgeExpired]) and are
    counted by [Cache.Len].
  - Keys must be hashable and equal to themselves: interface keys holding
    unhashable dynamic types cause [Cache.Lookup] to panic, and keys
    containing NaN never match themselves, leaking unevictable entries until
    [Cache.Reset].
  - Cached values are returned by reference: callers must treat returned
    values as read-only.

# Why It Matters

  - Reduces repeated network, database, or compute cost for hot keys.
  - Improves throughput in high-concurrency workloads by collapsing duplicate
    calls.
  - Keeps memory usage predictable with bounded capacity.

# Usage

The value type V is inferred from the lookup function, so Lookup returns
typed values with no assertions:

	cache := sfcache.New(func(ctx context.Context, key string) (*Customer, error) {
	    return fetchCustomer(ctx, key)
	}, 256, 5*time.Minute)

	customer, err := cache.Lookup(ctx, "customer:123")
	if err != nil {
	    return err
	}
	_ = customer // *Customer

Optional behaviors are enabled with options:

	cache := sfcache.New(lookupFn, 256, 5*time.Minute,
	    sfcache.WithTTLFunc(func(key string, c *Customer) time.Duration {
	        return c.TTL // freshness is a property of the data
	    }),
	    sfcache.WithStaleIfError[string, *Customer](10*time.Minute),
	)

Example applications in this repository include:
  - github.com/tecnickcom/gogen/pkg/awssecretcache
  - github.com/tecnickcom/gogen/pkg/dnscache
*/
package sfcache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ErrNilLookupFunc is returned by [Cache.Lookup] when the cache was
// constructed with a nil lookup function.
var ErrNilLookupFunc = errors.New("sfcache: the lookup function is nil")

// ErrLookupAborted is returned by [Cache.Lookup] when the caller's context
// ends while waiting for an in-flight lookup, or when the context has already
// ended before an external lookup would start. It wraps the context error, so
// errors.Is with [context.Canceled] or [context.DeadlineExceeded] keeps
// working.
var ErrLookupAborted = errors.New("sfcache: lookup aborted by context")

// LookupFunc is the generic function signature for external lookup calls.
type LookupFunc[K comparable, V any] func(ctx context.Context, key K) (V, error)

// entry stores cached value state for a single key.
// Entries are immutable once stored: updates replace the whole entry,
// so an *entry read under lock can be safely used after the lock is released.
type entry[V any] struct {
	// wait for each duplicate lookup call for the same key.
	// It is owned by the producing flight, which is the only one allowed to
	// close it (in fetch's defer); waiters only ever receive from it.
	wait chan struct{}

	// err is the error returned by the external lookup.
	err error

	// expireAt is the expiration deadline (monotonic clock).
	// The zero Time marks the entry as already expired
	// (in-flight placeholders, errors, and revived stale entries).
	expireAt time.Time

	// staleUntil is the deadline until which val may be served after a
	// failed refresh (see [WithStaleIfError]). The zero Time means no stale
	// value is available.
	staleUntil time.Time

	// val is the value associated with the key. On in-flight placeholders
	// with a non-zero staleUntil, it carries the last known good value.
	val V
}

// usable reports whether the entry holds a completed lookup outcome that the
// caller can return: a non-expired value or, for a caller that awaited this
// entry's flight, any completed outcome (even if already expired, e.g. with
// ttl <= 0 or a non-cached error).
func (e *entry[V]) usable(waited bool) bool {
	return e.wait == nil && (waited || time.Now().Before(e.expireAt))
}

// Cache is a generic, size-bounded single-flight cache with TTL expiration.
type Cache[K comparable, V any] struct {
	// keymap maps a key name to an item.
	keymap map[K]*entry[V]

	// lookupFn is the function performing the external lookup call.
	lookupFn LookupFunc[K, V]

	// ttlFn optionally computes a per-entry TTL (see [WithTTLFunc]).
	ttlFn TTLFunc[K, V]

	// mux is the mutex for the cache.
	mux *sync.RWMutex

	// ttl is the default time-to-live for the items.
	ttl time.Duration

	// maxStale bounds how long past its expiration a value may be served
	// when a refresh fails (see [WithStaleIfError]). Zero disables it.
	maxStale time.Duration

	// size is the maximum size of the cache (min = 1).
	size int
}

// New constructs a single-flight cache with the specified lookup function, max entries, and time-to-live.
// If lookupFn is nil, a default function is used that always fails with [ErrNilLookupFunc].
// Capacity defaults to 1 if size <= 0.
// A ttl <= 0 disables value caching while still coalescing duplicate in-flight
// requests for the same key (unless overridden per entry via [WithTTLFunc]).
// The default behavior can be customized with options such as [WithTTLFunc]
// and [WithStaleIfError].
func New[K comparable, V any](lookupFn LookupFunc[K, V], size int, ttl time.Duration, opts ...Option[K, V]) *Cache[K, V] {
	if lookupFn == nil {
		lookupFn = func(_ context.Context, _ K) (V, error) {
			var zero V

			return zero, ErrNilLookupFunc
		}
	}

	if size <= 0 {
		size = 1
	}

	c := &Cache[K, V]{
		lookupFn: lookupFn,
		mux:      &sync.RWMutex{},
		ttl:      ttl,
		size:     size,
		keymap:   make(map[K]*entry[V], size),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Len returns the current number of entries in the cache, including expired
// entries that have not been evicted yet and in-flight lookup placeholders.
func (c *Cache[K, V]) Len() int {
	c.mux.RLock()
	defer c.mux.RUnlock()

	return len(c.keymap)
}

// Reset clears all entries from the cache,
// including in-flight lookups, whose results will not be cached.
func (c *Cache[K, V]) Reset() {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.keymap = make(map[K]*entry[V], c.size)
}

// Remove deletes the cache entry for the specified key.
// If a lookup for the key is in flight, its result will not be cached.
func (c *Cache[K, V]) Remove(key K) {
	c.mux.Lock()
	defer c.mux.Unlock()

	delete(c.keymap, key)
}

// PurgeExpired removes all expired completed entries from the cache and
// returns the number of entries removed. In-flight lookups are not affected.
// NOTE: revived stale values (see [WithStaleIfError]) are stored as expired
// entries and are therefore purged too, forfeiting stale protection for
// their keys until the next successful lookup.
func (c *Cache[K, V]) PurgeExpired() int {
	c.mux.Lock()
	defer c.mux.Unlock()

	cuttime := time.Now()
	purged := 0

	for h, d := range c.keymap {
		if (d.wait == nil) && d.expireAt.Before(cuttime) {
			delete(c.keymap, h)

			purged++
		}
	}

	return purged
}

// Lookup retrieves the value for a key, performing single-flight deduplication for concurrent requests.
// Returns cached value if not expired; coalesces duplicate in-flight requests; evicts old/expired entries on capacity.
// Only successful values are cached for the TTL: errors are not cached (no negative caching), so every error triggers a fresh lookup on the next call.
// With [WithStaleIfError], a failed refresh may instead return the last known good value with a nil error.
// If the entry is removed (e.g. via [Cache.Remove] or [Cache.Reset]) while a lookup is in flight, the result is returned to its callers but not cached.
func (c *Cache[K, V]) Lookup(ctx context.Context, key K) (V, error) {
	// Fast path: a fresh cached value only requires the read lock.
	if item, ok := c.fresh(key); ok {
		return item.val, item.err
	}

	return c.lookupSlow(ctx, key)
}

// fresh returns the entry for the given key if it holds a non-expired
// completed value.
// The read lock is released via defer because this is the first map access
// for the key: an unhashable key (e.g. an interface holding a slice) panics
// here, and the lock must not leak with the panic. Later map accesses reuse a
// key that has already hashed successfully.
func (c *Cache[K, V]) fresh(key K) (*entry[V], bool) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	item, ok := c.keymap[key]

	return item, ok && item.usable(false)
}

// lookupSlow coalesces onto an in-flight lookup for the key, or performs the
// external lookup when the cache holds no usable entry.
func (c *Cache[K, V]) lookupSlow(ctx context.Context, key K) (V, error) {
	c.mux.Lock()

	waited := false

	for {
		item, ok := c.keymap[key]

		if !ok {
			// No entry: this caller performs the external lookup.
			return c.fetch(ctx, key)
		}

		if item.usable(waited) {
			c.mux.Unlock()
			return item.val, item.err
		}

		if item.wait == nil {
			// Expired completed entry: this caller refreshes it.
			return c.fetch(ctx, key)
		}

		// Another external lookup is already in progress,
		// waiting for completion and return values from cache.
		c.mux.Unlock()

		// Wait until the external lookup is completed,
		// or the Context is canceled.
		// NOTE: when both channels are ready at the same instant, select
		// picks pseudo-randomly: a waiter whose context ends as the flight
		// completes may observe either outcome. This nondeterminism is
		// inherent and documented in the package comment.
		select {
		case <-ctx.Done():
			// A waiter must never close item.wait: it is owned by the
			// in-flight goroutine, which closes it when lookupFn returns.
			// Closing it here would double-close and panic.
			var zero V

			return zero, fmt.Errorf("%w: %w", ErrLookupAborted, ctx.Err())
		case <-item.wait:
		}

		// Re-evaluate the cache state from scratch: the entry may have
		// completed, been removed (Remove, Reset, panic recovery, or a
		// canceled lookup), or been replaced by a new in-flight lookup.
		waited = true

		c.mux.Lock()
	}
}

// fetch performs the external lookup as the single producer for the key,
// publishing the in-flight placeholder and then the final result.
// NOTE: it must be called with the write lock held; it releases it.
func (c *Cache[K, V]) fetch(ctx context.Context, key K) (V, error) {
	var zero V

	ctxErr := ctx.Err()
	if ctxErr != nil {
		// Never start an external lookup with an already-ended context.
		c.mux.Unlock()
		return zero, fmt.Errorf("%w: %w", ErrLookupAborted, ctxErr)
	}

	wait := make(chan struct{})
	finalized := false

	defer func() {
		if !finalized {
			// lookupFn panicked: remove the in-flight placeholder so waiters
			// observe a terminal state (missing entry) instead of busy-spinning
			// on a closed wait channel. The panic propagates to the caller.
			c.abortFlight(key, wait)
		}

		close(wait)
	}()

	// Carry the last known good value through the flight, so that a failed
	// refresh can serve it (see [WithStaleIfError]).
	staleVal, staleUntil := c.staleFrom(c.keymap[key])

	c.set(key, zero, nil, wait)

	if !staleUntil.IsZero() {
		// Replace (not mutate) the just-installed placeholder:
		// entries are immutable once stored.
		c.keymap[key] = &entry[V]{wait: wait, val: staleVal, staleUntil: staleUntil}
	}

	c.mux.Unlock()

	val, err := c.lookupFn(ctx, key)

	val, err = c.publish(ctx, key, wait, val, err)

	finalized = true

	return val, err
}

// staleFrom extracts the last known good value and its serving deadline from
// the entry being replaced by a refresh flight, when stale-if-error is
// enabled. A previously revived stale entry keeps its original deadline; a
// regular value entry gets its expiration plus maxStale; error residue and
// missing entries yield no stale value.
func (c *Cache[K, V]) staleFrom(old *entry[V]) (V, time.Time) {
	if (c.maxStale <= 0) || (old == nil) || (old.err != nil) {
		var zero V

		return zero, time.Time{}
	}

	if !old.staleUntil.IsZero() {
		return old.val, old.staleUntil
	}

	return old.val, old.expireAt.Add(c.maxStale)
}

// abortFlight removes the in-flight placeholder identified by its wait
// channel, leaving a terminal state (missing entry) for waiters to observe.
func (c *Cache[K, V]) abortFlight(key K, wait chan struct{}) {
	c.mux.Lock()

	if item, ok := c.keymap[key]; ok && (item.wait == wait) {
		delete(c.keymap, key)
	}

	c.mux.Unlock()
}

// publish stores the outcome of the lookup flight identified by its wait
// channel and returns the outcome the flight's caller must receive.
// If the placeholder was removed (Remove or Reset) mid-flight, the outcome is
// discarded: it is still returned to the flight's caller but not cached.
// If the lookup failed while a stale value is available (see
// [WithStaleIfError]), the stale value is revived and returned with a nil
// error, regardless of the failure's cause: serving the last known good value
// beats retrying an upstream that may be hanging.
// Otherwise, if the lookup failed with the producing caller's own context
// error, the placeholder is removed instead of publishing the context-induced
// error, so that a coalesced waiter with a live context retries the lookup.
// NOTE: the context check uses errors.Is against ctx.Err(), which matches
// sentinel identity rather than provenance: a lookup error wrapping an
// unrelated context.DeadlineExceeded can be misclassified when the producing
// context has also ended, turning an error residue into a waiter retry.
// Remaining genuine errors are published and shared with waiters as usual.
func (c *Cache[K, V]) publish(ctx context.Context, key K, wait chan struct{}, val V, err error) (V, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	item, ok := c.keymap[key]
	if !ok || (item.wait != wait) {
		// The placeholder was removed (Remove or Reset) mid-flight:
		// the result is returned to this caller but not cached.
		return val, err
	}

	if err == nil {
		c.set(key, val, nil, nil)

		return val, nil
	}

	if time.Now().Before(item.staleUntil) {
		// Stale-if-error: revive the last known good value carried by the
		// placeholder instead of publishing the error. The revived entry
		// stays expired (zero expireAt), so the next call attempts a fresh
		// lookup again. Reclaim any capacity excess first: the placeholder
		// for this key is never evicted, so the revived entry cannot be its
		// own victim.
		c.makeRoom(key)

		c.keymap[key] = &entry[V]{val: item.val, staleUntil: item.staleUntil}

		return item.val, nil
	}

	if (ctx.Err() != nil) && errors.Is(err, ctx.Err()) {
		delete(c.keymap, key)

		return val, err
	}

	c.set(key, val, err, nil)

	return val, err
}

// set adds or updates the cache entry for the given key with the provided value.
// If the cache is over its capacity, it frees up space by removing expired or
// old entries, also reclaiming any excess accumulated while more than `size`
// distinct keys were in flight at once.
// If the key already exists in the cache, it will update the entry with the new value.
// NOTE: this is not thread-safe, it should be called within a mutex lock.
func (c *Cache[K, V]) set(key K, val V, err error, wait chan struct{}) {
	c.makeRoom(key)

	var expireAt time.Time

	if (err == nil) && (wait == nil) {
		// Only successful completed lookups are cached for the TTL
		// (including legitimate nil values): errors are never cached
		// (no negative caching) and in-flight placeholders (wait != nil)
		// must stay expired so duplicate callers wait on the channel.
		expireAt = time.Now().Add(c.entryTTL(key, val))
	}

	c.keymap[key] = &entry[V]{
		wait:     wait,
		err:      err,
		expireAt: expireAt,
		val:      val,
	}
}

// entryTTL returns the time-to-live for a new entry: the per-entry override
// when a TTL function is configured and returns a positive duration (see
// [WithTTLFunc]), or the cache-wide default otherwise.
func (c *Cache[K, V]) entryTTL(key K, val V) time.Duration {
	if c.ttlFn != nil {
		if d := c.ttlFn(key, val); d > 0 {
			return d
		}
	}

	return c.ttl
}

// makeRoom evicts entries until the cache fits its capacity target for
// storing the given key. Eviction can fall short when the remainder is made
// of in-flight placeholders, which are never evicted.
// NOTE: this is not thread-safe, it should be called within a mutex lock.
func (c *Cache[K, V]) makeRoom(key K) {
	target := c.size

	if _, ok := c.keymap[key]; !ok {
		// make room for the new entry
		target--
	}

	for (len(c.keymap) > target) && c.evict() {
	}
}

// evict removes either the first expired entry found or the oldest entry by
// expiration deadline from the cache, reporting whether an entry was removed.
// In-flight placeholders (entries with a non-nil wait channel) are never evicted,
// as removing them would break single-flight deduplication.
// NOTE: this is not thread-safe, it should be called within a mutex lock.
func (c *Cache[K, V]) evict() bool {
	cuttime := time.Now()
	found := false

	var (
		oldest    time.Time
		oldestkey K
	)

	for h, d := range c.keymap {
		if d.wait != nil {
			// skip in-flight placeholders
			continue
		}

		if d.expireAt.Before(cuttime) {
			delete(c.keymap, h)
			return true
		}

		if !found || d.expireAt.Before(oldest) {
			oldest = d.expireAt
			oldestkey = h
			found = true
		}
	}

	if found {
		delete(c.keymap, oldestkey)
	}

	return found
}
