package filter

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

const (
	// FieldNameSeparator is the separator for Rule fields.
	FieldNameSeparator = "."
)

// errFieldNotFound is returned when a specified field is not found in a struct.
var errFieldNotFound = errors.New("field not found")

// reflectPath stores a dot path (for example, "address.country") as field indices (for example, []int{2,1}) usable with reflect.Value.Field.
type reflectPath []int

// fieldGetter retrieves field values from objects based on their field paths.
type fieldGetter struct {
	fieldTag string
	maxDepth uint
	cache    fieldCache
}

// resolvePath returns the field-index path for the dot-separated selector within
// type t, resolving and caching it on first use. It is called once per type per
// Apply for concrete element slices, and per element only when the slice element
// type is an interface (so the concrete type varies per element).
//
// A non-empty path targeting a missing field returns an errFieldNotFound-wrapped
// error, which callers treat as a non-match rather than a hard failure.
func (r *fieldGetter) resolvePath(t reflect.Type, path string) (reflectPath, error) {
	// Reject over-deep selectors before resolving or caching them. This bounds both the
	// O(depth) resolution cost and, for recursive element types (whose valid paths are
	// unbounded), the size of the path cache.
	if depth := uint(strings.Count(path, FieldNameSeparator) + 1); depth > r.maxDepth {
		return nil, fmt.Errorf("%w: field path too deep: got %d max is %d", ErrInvalidFilter, depth, r.maxDepth)
	}

	if rPath, ok := r.cache.Get(t, path); ok {
		return rPath, nil
	}

	rPath, err := r.getFieldPath(t, strings.Split(path, FieldNameSeparator))
	if err != nil {
		return nil, err
	}

	r.cache.Set(t, path, rPath)

	return rPath, nil
}

// getFieldPath constructs the reflectPath for the given type and field names.
func (r *fieldGetter) getFieldPath(t reflect.Type, fieldNames []string) (reflectPath, error) {
	fieldPath := make(reflectPath, 0, len(fieldNames))

	for len(fieldNames) > 0 {
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}

		if t.Kind() != reflect.Struct {
			return nil, fmt.Errorf("%w: fields of elements of type %s are not supported", ErrInvalidFilter, t)
		}

		field, err := r.getStructField(t, fieldNames[0])
		if err != nil {
			return nil, err
		}

		fieldPath = append(fieldPath, field.Index...)

		fieldNames = fieldNames[1:]
		t = field.Type
	}

	return fieldPath, nil
}

// getStructField retrieves the struct field by name or tag.
//
// Any path segment that resolves to an unexported field is rejected: reflection cannot read
// such a value (Value.Interface panics), so it can never match, and reporting it here makes
// that deterministic instead of an eval-time error that only a non-empty slice would surface.
// An exported field promoted through an unexported embedded type keeps PkgPath == "" and stays
// selectable via its promoted name; naming the unexported embed explicitly (e.g. "inner.Field"
// rather than the promoted "Field") is rejected, because the segment itself is unexported.
func (r *fieldGetter) getStructField(t reflect.Type, name string) (reflect.StructField, error) {
	field, ok := r.lookupStructField(t, name)
	if !ok {
		if r.fieldTag == "" {
			return reflect.StructField{}, fmt.Errorf("field %s.%s: %w", t, name, errFieldNotFound)
		}

		return reflect.StructField{}, fmt.Errorf("field of %s with tag %s=%s: %w", t, r.fieldTag, name, errFieldNotFound)
	}

	if field.PkgPath != "" {
		return reflect.StructField{}, fmt.Errorf("%w: field %s.%s is unexported and cannot be read", ErrInvalidFilter, t, field.Name)
	}

	return field, nil
}

// lookupStructField finds a struct field by name, or by the configured tag when set.
func (r *fieldGetter) lookupStructField(t reflect.Type, name string) (reflect.StructField, bool) {
	if r.fieldTag == "" {
		return t.FieldByName(name)
	}

	return r.lookupFieldByTag(t, name)
}

// lookupFieldByTag looks up a struct field by its tag value.
func (r *fieldGetter) lookupFieldByTag(t reflect.Type, tagValue string) (reflect.StructField, bool) {
	for _, field := range reflect.VisibleFields(t) {
		actualValue := field.Tag.Get(r.fieldTag)
		actualValue = strings.Split(actualValue, ",")[0]

		if actualValue == tagValue {
			return field, true
		}
	}

	return reflect.StructField{}, false
}
