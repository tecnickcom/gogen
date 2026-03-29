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
	cache    fieldCache
}

// GetFieldValue resolves the dot-separated field path within obj, with caching and nil handling.
// Empty path returns the root object; missing fields return error, not nil.
func (r *fieldGetter) GetFieldValue(obj any, path string) (any, error) {
	// empty path means the root object
	if path == "" {
		return obj, nil
	}

	if obj == nil {
		return nil, errors.New("cannot get a field of a nil object")
	}

	tElement := reflect.TypeOf(obj)

	rPath, ok := r.cache.Get(tElement, path)
	if !ok {
		var err error

		pathParts := strings.Split(path, FieldNameSeparator)

		rPath, err = r.getFieldPath(tElement, pathParts)
		if err != nil {
			return nil, err
		}

		r.cache.Set(tElement, path, rPath)
	}

	value := reflect.ValueOf(obj)
	for _, fieldIndex := range rPath {
		value = reflect.Indirect(value)
		value = value.Field(fieldIndex)
	}

	if !value.CanInterface() {
		return nil, fmt.Errorf("%s cannot be interfaced", value.Type())
	}

	return value.Interface(), nil
}

// getFieldPath constructs the reflectPath for the given type and field names.
func (r *fieldGetter) getFieldPath(t reflect.Type, fieldNames []string) (reflectPath, error) {
	fieldPath := make(reflectPath, 0, len(fieldNames))

	for len(fieldNames) > 0 {
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}

		if t.Kind() != reflect.Struct {
			return nil, fmt.Errorf("fields of elements of type %s are not supported", t)
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
func (r *fieldGetter) getStructField(t reflect.Type, name string) (reflect.StructField, error) {
	if r.fieldTag == "" {
		field, ok := t.FieldByName(name)
		if !ok {
			return reflect.StructField{}, fmt.Errorf("field %s.%s: %w", t, name, errFieldNotFound)
		}

		return field, nil
	}

	field, ok := r.lookupFieldByTag(t, name)
	if !ok {
		return reflect.StructField{}, fmt.Errorf("field of %s with tag %s=%s: %w", t, r.fieldTag, name, errFieldNotFound)
	}

	return field, nil
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
