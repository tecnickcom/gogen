package filter

import (
	"reflect"
)

type lte struct {
	ref interface{}
}

func newLTE(r interface{}) (Evaluator, error) {
	var err error

	r, err = convertNumericValue(r)
	if err != nil {
		return nil, err
	}

	return &lte{ref: r}, nil
}

// Evaluate returns whether the actual value is less than the reference.
// It converts numerical values implicitly before comparison.
// Returns the lengths comparison for Array, Map, Slice or String.
// Returns true if the value is nil.
func (e *lte) Evaluate(v interface{}) bool {
	v = convertValue(v)

	if isNil(v) {
		return true
	}

	val := reflect.ValueOf(v)
	ref := reflect.ValueOf(e.ref).Float()

	//nolint:exhaustive
	switch val.Kind() {
	case reflect.Float64:
		return val.Float() <= ref
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return val.Len() <= int(ref)
	}

	return false
}
