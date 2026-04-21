package filter

import (
	"reflect"
)

// gte is an Evaluator that checks if a value is greater than or equal to a reference.
type gte struct {
	ref float64
}

// newGTE constructs a greater-than-or-equal evaluator from a reference numeric value.
// Returns error if r cannot be converted to float64.
func newGTE(r any) (Evaluator, error) {
	v, err := convertFloatValue(r)
	if err != nil {
		return nil, err
	}

	return &gte{ref: v}, nil
}

// Evaluate returns true if the numeric value is >= reference, or collection length is >= reference for arrays/maps/slices/strings.
func (e *gte) Evaluate(v any) bool {
	v = convertValue(v)

	if isNil(v) {
		return false
	}

	val := reflect.ValueOf(v)

	//nolint:exhaustive
	switch val.Kind() {
	case reflect.Float64:
		return val.Float() >= e.ref
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return val.Len() >= int(e.ref)
	}

	return false
}
