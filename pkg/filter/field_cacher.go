package filter

import (
	"reflect"
	"sync"
)

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
	cache sync.Map // fieldCacheKey -> reflectPath
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

// Set caches the reflectPath for a field by its type and path.
func (c *fieldCache) Set(t reflect.Type, fieldPath string, path reflectPath) {
	c.cache.Store(fieldCacheKey{t: t, fieldPath: fieldPath}, path)
}
