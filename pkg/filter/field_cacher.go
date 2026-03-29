package filter

import "reflect"

// pathByField stores reflectPath by field name.
type pathByField map[string]reflectPath

// fieldByType stores pathByField by struct type.
type fieldByType map[string]pathByField

// fieldCache caches resolved reflection paths by type and field selector.
type fieldCache struct {
	cache fieldByType
}

// Get retrieves the cached reflectPath for a field given its type and path.
// Returns (nil, false) if not cached.
func (c *fieldCache) Get(t reflect.Type, fieldPath string) (reflectPath, bool) {
	fields := c.getFieldsMap(t)
	path, ok := fields[fieldPath]

	return path, ok
}

// Set caches the reflectPath for a field by its type and path.
func (c *fieldCache) Set(t reflect.Type, fieldPath string, path reflectPath) {
	fields := c.getFieldsMap(t)
	fields[fieldPath] = path
}

// getFieldsMap retrieves or lazy-initializes the pathByField map for the given type.
func (c *fieldCache) getFieldsMap(t reflect.Type) pathByField {
	if c.cache == nil {
		c.cache = make(fieldByType)
	}

	tKey := t.String()

	fields, ok := c.cache[tKey]
	if !ok {
		fields = make(pathByField)
		c.cache[tKey] = fields
	}

	return fields
}
