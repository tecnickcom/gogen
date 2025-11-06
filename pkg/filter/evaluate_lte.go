package filter

import (
	"reflect"
)

// lte is an Evaluator that checks if a value is less than or equal to a reference.
type lte struct {
	ref float64
}

// newLTE returns an Evaluator that checks if a value is less than or equal to the reference.
func newLTE(r any) (Evaluator, error) {
	v, err := convertFloatValue(r)
	if err != nil {
		return nil, err
	}

	return &lte{ref: v}, nil
}

// Evaluate returns whether the actual value is less than or equal the reference.
// It converts numerical values implicitly before comparison.
// Returns the lengths comparison for Array, Map, Slice or String.
// Returns false if the value is nil.
func (e *lte) Evaluate(v any) bool {
	v = convertValue(v)

	if isNil(v) {
		return false
	}

	val := reflect.ValueOf(v)

	switch val.Kind() { //nolint:exhaustive
	case reflect.Float64:
		return val.Float() <= e.ref
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return val.Len() <= int(e.ref)
	}

	return false
}
