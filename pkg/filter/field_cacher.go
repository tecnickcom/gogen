package filter

import (
	"reflect"
	"sync"
	"sync/atomic"
)

// defaultFieldCacheMaxEntries bounds how many resolved paths a single Processor caches. A
// self-referential element type has an unbounded number of valid selectors (only their length
// is capped, by WithMaxFieldDepth), so without a ceiling a long-lived shared Processor fed
// distinct untrusted filters would grow without limit. Past the ceiling, resolution still
// works — it is just recomputed per use instead of cached.
const defaultFieldCacheMaxEntries = 1 << 14 // 16384

// fieldCacheKey uniquely identifies a resolved field path.
// Keying by reflect.Type (comparable and unique per type) instead of its String()
// avoids collisions between identically named types from different packages.
type fieldCacheKey struct {
	t         reflect.Type
	fieldPath string
}

// fieldCache caches resolved reflection paths by type and field selector.
// It is safe for concurrent use, so a shared Processor can Apply from multiple goroutines.
type fieldCache struct {
	cache      sync.Map     // fieldCacheKey -> reflectPath
	count      atomic.Int64 // number of entries stored, to bound growth
	maxEntries int64        // entry ceiling; 0 selects defaultFieldCacheMaxEntries
}

// Get retrieves the cached reflectPath for a field given its type and path.
// Returns (nil, false) if not cached.
func (c *fieldCache) Get(t reflect.Type, fieldPath string) (reflectPath, bool) {
	v, ok := c.cache.Load(fieldCacheKey{t: t, fieldPath: fieldPath})
	if !ok {
		return nil, false
	}

	path, ok := v.(reflectPath)

	return path, ok
}

// Set caches the reflectPath for a field by its type and path, up to the entry ceiling.
func (c *fieldCache) Set(t reflect.Type, fieldPath string, path reflectPath) {
	ceiling := c.maxEntries
	if ceiling == 0 {
		ceiling = defaultFieldCacheMaxEntries
	}

	// Skip once full: correctness does not depend on the cache, only speed, so a miss beyond
	// the ceiling simply re-resolves. Counting only newly stored keys keeps the bound honest
	// under concurrent stores of the same key.
	if c.count.Load() >= ceiling {
		return
	}

	if _, loaded := c.cache.LoadOrStore(fieldCacheKey{t: t, fieldPath: fieldPath}, path); !loaded {
		c.count.Add(1)
	}
}
