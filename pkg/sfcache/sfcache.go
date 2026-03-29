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

	// expireAt is the expiration time in seconds elapsed since January 1, 1970 UTC.
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
//
//nolint:gocognit
func (c *Cache[K]) Lookup(ctx context.Context, key K) (any, error) {
	c.mux.Lock()
	item, ok := c.keymap[key]

	//nolint:nestif
	if ok {
		if item.expireAt > time.Now().UTC().Unix() {
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
					defer close(item.wait)
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
	defer close(wait)

	c.set(key, nil, nil, wait)
	c.mux.Unlock()

	val, err := c.lookupFn(ctx, key)

	c.mux.Lock()
	c.set(key, val, err, nil)
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

	var now int64

	if val != nil {
		now = time.Now().UTC().Add(c.ttl).Unix()
	}

	c.keymap[key] = &entry{
		wait:     wait,
		err:      err,
		expireAt: now,
		val:      val,
	}
}

// evict removes either the oldest entry or the first expired one from the cache.
// NOTE: this is not thread-safe, it should be called within a mutex lock.
func (c *Cache[K]) evict() {
	cuttime := time.Now().UTC().Unix()
	oldest := int64(1<<63 - 1)

	var oldestkey K

	for h, d := range c.keymap {
		if d.expireAt < cuttime {
			delete(c.keymap, h)
			return
		}

		if d.expireAt < oldest {
			oldest = d.expireAt
			oldestkey = h
		}
	}

	delete(c.keymap, oldestkey)
}
