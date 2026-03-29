package filter

import (
	"reflect"
)

// lte is an Evaluator that checks if a value is less than or equal to a reference.
type lte struct {
	ref float64
}

// newLTE constructs a less-than-or-equal evaluator from a reference numeric value.
// Returns error if r cannot be converted to float64.
func newLTE(r any) (Evaluator, error) {
	v, err := convertFloatValue(r)
	if err != nil {
		return nil, err
	}

	return &lte{ref: v}, nil
}

// Evaluate returns true if the numeric value is <= reference, or collection length is <= reference for arrays/maps/slices/strings.
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
