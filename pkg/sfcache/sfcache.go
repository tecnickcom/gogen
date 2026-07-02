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
  - a time-to-live (`ttl`) for successful values.

On [Cache.Lookup]:

 1. If a non-expired entry exists, the cached value is returned immediately.
 2. If the key is being resolved by another goroutine, duplicate callers wait
    and receive that same result (single-flight behavior).
 3. On miss or expiry, one lookup function call is executed and its result is
    stored in cache.
 4. If cache capacity is reached, eviction removes an expired entry first, or
    otherwise the oldest entry by expiration timestamp.

# Key Features

  - Fixed-size local cache with explicit capacity to avoid unbounded memory
    growth.
  - Internal synchronization for safe concurrent access without external locks.
  - Single-flight request collapsing for duplicate in-flight lookups.
  - TTL-based freshness with automatic refresh on next miss after expiry.
  - Explicit cache control via [Cache.Remove] and [Cache.Reset].

# Why It Matters

  - Reduces repeated network, database, or compute cost for hot keys.
  - Improves throughput in high-concurrency workloads by collapsing duplicate
    calls.
  - Keeps memory usage predictable with bounded capacity.

# Usage

	cache := sfcache.New(func(ctx context.Context, key string) (any, error) {
	    return fetchRemoteValue(ctx, key)
	}, 256, 5*time.Minute)

	v, err := cache.Lookup(ctx, "customer:123")
	if err != nil {
	    return err
	}
	_ = v

Example applications in this repository include:
  - github.com/tecnickcom/gogen/pkg/awssecretcache
  - github.com/tecnickcom/gogen/pkg/dnscache
*/
package sfcache

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// LookupFunc is the generic function signature for external lookup calls.
type LookupFunc[K comparable] func(ctx context.Context, key K) (any, error)

// entry stores cached value state for a single key.
type entry struct {
	// wait for each duplicate lookup call for the same key.
	wait chan struct{}

	// err is the error returned by the external lookup.
	err error

	// expireAt is the expiration time in nanoseconds elapsed since January 1, 1970 UTC.
	// A zero value marks the entry as already expired (in-flight placeholders and errors).
	expireAt int64

	// val is the value associated with the key.
	val any
}

// Cache is a generic, size-bounded single-flight cache with TTL expiration.
type Cache[K comparable] struct {
	// keymap maps a key name to an item.
	keymap map[K]*entry

	// lookupFn is the function performing the external lookup call.
	lookupFn LookupFunc[K]

	// mux is the mutex for the cache.
	mux *sync.RWMutex

	// ttl is the time-to-live for the items.
	ttl time.Duration

	// size is the maximum size of the cache (min = 1).
	size int
}

// New constructs a single-flight cache with the specified lookup function, max entries, and time-to-live.
// Capacity defaults to 1 if size <= 0; duplicate in-flight requests for the same key wait for the first result.
func New[K comparable](lookupFn LookupFunc[K], size int, ttl time.Duration) *Cache[K] {
	if size <= 0 {
		size = 1
	}

	return &Cache[K]{
		lookupFn: lookupFn,
		mux:      &sync.RWMutex{},
		ttl:      ttl,
		size:     size,
		keymap:   make(map[K]*entry, size),
	}
}

// Len returns the current number of entries in the cache.
func (c *Cache[K]) Len() int {
	c.mux.RLock()
	defer c.mux.RUnlock()

	return len(c.keymap)
}

// Reset clears all entries from the cache.
func (c *Cache[K]) Reset() {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.keymap = make(map[K]*entry, c.size)
}

// Remove deletes the cache entry for the specified key.
func (c *Cache[K]) Remove(key K) {
	c.mux.Lock()
	defer c.mux.Unlock()

	delete(c.keymap, key)
}

// Lookup retrieves the value for a key, performing single-flight deduplication for concurrent requests.
// Returns cached value if not expired; coalesces duplicate in-flight requests; evicts old/expired entries on capacity.
// Only successful values are cached for the TTL: errors are not cached (no negative caching), so every error triggers a fresh lookup on the next call.
//
//nolint:gocognit,gocyclo,cyclop
func (c *Cache[K]) Lookup(ctx context.Context, key K) (any, error) {
	c.mux.Lock()
	item, ok := c.keymap[key]

	//nolint:nestif
	if ok {
		if item.expireAt > time.Now().UTC().UnixNano() {
			c.mux.Unlock()
			return item.val, item.err
		}

		if item.wait != nil {
			// Another external lookup is already in progress,
			// waiting for completion and return values from cache.
			c.mux.Unlock()

			for {
				// Wait until the external lookup is completed,
				// or the Context is canceled.
				select {
				case <-ctx.Done():
					// A waiter must never close item.wait: it is owned by the
					// in-flight goroutine, which closes it when lookupFn returns.
					// Closing it here would double-close and panic.
					return nil, fmt.Errorf("context canceled: %w", ctx.Err())
				case <-item.wait:
				}

				c.mux.RLock()
				item, ok = c.keymap[key]
				c.mux.RUnlock()

				if !ok {
					// The cache entry was removed during the wait.
					break
				}

				if item.wait != nil {
					// The cache entry was updated during the wait.
					// This should not happen in real world scenarios,
					// but it's good to have it covered.
					continue
				}

				return item.val, item.err
			}

			// The cache entry was removed during the wait,
			// move on to perform a new lookup.
			c.mux.Lock()
		}
	}

	wait := make(chan struct{})
	finalized := false

	defer func() {
		if !finalized {
			// lookupFn panicked: remove the in-flight placeholder so waiters
			// observe a terminal state (missing entry) instead of busy-spinning
			// on a closed wait channel. The panic propagates to the caller.
			c.mux.Lock()

			if item, ok := c.keymap[key]; ok && (item.wait == wait) {
				delete(c.keymap, key)
			}

			c.mux.Unlock()
		}

		close(wait)
	}()

	c.set(key, nil, nil, wait)
	c.mux.Unlock()

	val, err := c.lookupFn(ctx, key)

	c.mux.Lock()
	c.set(key, val, err, nil)

	finalized = true
	c.mux.Unlock()

	return val, err
}

// set adds or updates the cache entry for the given key with the provided value.
// If the cache is full, it will free up space by removing expired or old entries.
// If the key already exists in the cache, it will update the entry with the new value.
// NOTE: this is not thread-safe, it should be called within a mutex lock.
func (c *Cache[K]) set(key K, val any, err error, wait chan struct{}) {
	if len(c.keymap) >= c.size {
		if _, ok := c.keymap[key]; !ok {
			// free up space for a new entry
			c.evict()
		}
	}

	var expireAt int64

	if (err == nil) && (wait == nil) {
		// Only successful completed lookups are cached for the TTL
		// (including legitimate nil values): errors are never cached
		// (no negative caching) and in-flight placeholders (wait != nil)
		// must stay expired so duplicate callers wait on the channel.
		expireAt = time.Now().UTC().Add(c.ttl).UnixNano()
	}

	c.keymap[key] = &entry{
		wait:     wait,
		err:      err,
		expireAt: expireAt,
		val:      val,
	}
}

// evict removes either the oldest entry or the first expired one from the cache.
// In-flight placeholders (entries with a non-nil wait channel) are never evicted,
// as removing them would break single-flight deduplication.
// NOTE: this is not thread-safe, it should be called within a mutex lock.
func (c *Cache[K]) evict() {
	cuttime := time.Now().UTC().UnixNano()
	oldest := int64(1<<63 - 1)
	found := false

	var oldestkey K

	for h, d := range c.keymap {
		if d.wait != nil {
			// skip in-flight placeholders
			continue
		}

		if d.expireAt < cuttime {
			delete(c.keymap, h)
			return
		}

		if d.expireAt < oldest {
			oldest = d.expireAt
			oldestkey = h
			found = true
		}
	}

	if found {
		delete(c.keymap, oldestkey)
	}
}
