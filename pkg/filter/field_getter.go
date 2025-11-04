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

var errFieldNotFound = errors.New("field not found")

// reflectPath represents a field path (e.g. address.country) as the indices of the fields (e.g. [2,1]) that can be used with reflect.Value.Field(i int).
type reflectPath []int

type fieldGetter struct {
	fieldTag string
	cache    fieldCache
}

// GetFieldValue returns the value of obj's field, specified by its dot separated path.
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
