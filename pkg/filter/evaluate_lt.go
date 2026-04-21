package filter

import (
	"reflect"
)

// lt is an Evaluator that checks if a value is less than a reference.
type lt struct {
	ref float64
}

// newLT constructs a less-than evaluator from a reference numeric value.
// Returns error if r cannot be converted to float64.
func newLT(r any) (Evaluator, error) {
	v, err := convertFloatValue(r)
	if err != nil {
		return nil, err
	}

	return &lt{ref: v}, nil
}

// Evaluate returns true if the numeric value is less than reference, or collection length is less than reference for arrays/maps/slices/strings.
func (e *lt) Evaluate(v any) bool {
	v = convertValue(v)

	if isNil(v) {
		return false
	}

	val := reflect.ValueOf(v)

	//nolint:exhaustive
	switch val.Kind() {
	case reflect.Float64:
		return val.Float() < e.ref
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return val.Len() < int(e.ref)
	}

	return false
}
