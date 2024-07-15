/*
Package sfcache provides a simple, local, thread-safe, fixed-size, and
single-flight cache for expensive lookup calls.

This package is designed to improve the performance of expensive, slow, or
high-frequency function calls that retrieve data associated with a unique
identifier (key). It achieves this by caching previous values, eliminating the
need for repeated expensive requests.

The sfcache package offers a local in-memory cache with a configurable maximum
number of entries. The fixed-size nature of the cache ensures efficient memory
management and prevents excessive memory usage. Additionally, the cache is
thread-safe, allowing concurrent access without the need for external
synchronization. It efficiently handles concurrent requests by sharing results
from the first lookup, ensuring that only one request performs the expensive
call. This approach avoids unnecessary network load or resource starvation.
Duplicate calls for the same key will wait for the first call to complete and
return the same value.

Each cache entry has a time-to-live (TTL) value, which determines its
expiration. The cache also provides methods to force the removal of a specific
entry or reset the entire cache.

The sfcache package is ideal for any Go application that heavily relies on
expensive or slow lookups.

Example applications that can benefit from this package include:
  - github.com/Vonage/gosrvlib/pkg/awssecretcache
  - github.com/Vonage/gosrvlib/pkg/dnscache
*/
package sfcache

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// LookupFunc is the generic function signature for external lookup calls.
type LookupFunc func(ctx context.Context, key string) (any, error)

// entry represents a cache entry for a given key.
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

// Cache represents a cache for items.
type Cache struct {
	// keymap maps a key name to an item.
	keymap map[string]*entry

	// lookupFn is the function performing the external lookup call.
	lookupFn LookupFunc

	// mux is the mutex for the cache.
	mux *sync.RWMutex

	// ttl is the time-to-live for the items.
	ttl time.Duration

	// size is the maximum size of the cache (min = 1).
	size int
}

// New creates a new single-flight cache of the specified size and TTL.
// The lookup function performs the external call for each cache miss.
// The size parameter determines the maximum number of entries that can be cached (min = 1).
// If the size is less than or equal to zero, the cache will have a default size of 1.
// The ttl parameter specifies the time-to-live for each cached entry.
func New(lookupFn LookupFunc, size int, ttl time.Duration) *Cache {
	if size <= 0 {
		size = 1
	}

	return &Cache{
		lookupFn: lookupFn,
		mux:      &sync.RWMutex{},
		ttl:      ttl,
		size:     size,
		keymap:   make(map[string]*entry, size),
	}
}

// Len returns the number of items in the cache.
func (c *Cache) Len() int {
	c.mux.RLock()
	defer c.mux.RUnlock()

	return len(c.keymap)
}

// Reset clears the whole cache.
func (c *Cache) Reset() {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.keymap = make(map[string]*entry, c.size)
}

// Remove removes the cache entry for the specified key.
func (c *Cache) Remove(key string) {
	c.mux.Lock()
	defer c.mux.Unlock()

	delete(c.keymap, key)
}

// Lookup performs a lookup for the given key.
// Duplicate lookup calls for the same key will wait for the first lookup to complete (single-flight).
// This function uses a mutex lock to ensure thread safety.
// It also handles the case where the cache entry is removed or updated during the wait.
// The function returns the cached value if available; otherwise, it performs a new lookup.
// If the external lookup call is successful, it updates the cache with the newly obtained value.
//
//nolint:gocognit
func (c *Cache) Lookup(ctx context.Context, key string) (any, error) {
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
func (c *Cache) set(key string, val any, err error, wait chan struct{}) {
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
func (c *Cache) evict() {
	cuttime := time.Now().UTC().Unix()
	oldest := int64(1<<63 - 1)
	oldestkey := ""

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
