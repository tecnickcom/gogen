package filter

import (
	"reflect"
)

// gt evaluates whether a value is greater than a reference.
type gt struct {
	ref float64
}

// newGT constructs a greater-than evaluator from a reference numeric value.
// Returns error if r cannot be converted to float64.
func newGT(r any) (Evaluator, error) {
	v, err := convertFloatValue(r)
	if err != nil {
		return nil, err
	}

	return &gt{ref: v}, nil
}

// Evaluate returns true if the numeric value exceeds the reference, or collection length exceeds reference for arrays/maps/slices/strings.
func (e *gt) Evaluate(v any) bool {
	v = convertValue(v)

	if isNil(v) {
		return false
	}

	val := reflect.ValueOf(v)

	//nolint:exhaustive
	switch val.Kind() {
	case reflect.Float64:
		return val.Float() > e.ref
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return val.Len() > int(e.ref)
	}

	return false
}
