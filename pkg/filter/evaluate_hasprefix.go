package filter

import (
	"fmt"
	"reflect"
	"strings"
)

// evalHasPrefix is an Evaluator that checks if a string begins with a reference prefix.
type evalHasPrefix struct {
	ref string
}

// newHasPrefix returns an Evaluator that checks if a string begins with the reference prefix.
func newHasPrefix(r any) (Evaluator, error) {
	str, ok := r.(string)
	if !ok {
		return nil, fmt.Errorf("rule of type %s should have string value (got %v (%v))", TypeHasPrefix, r, reflect.TypeOf(r))
	}

	return &evalHasPrefix{ref: str}, nil
}

// Evaluate returns whether the input value begins with the reference string.
// It returns false if the input value is not a string.
func (e *evalHasPrefix) Evaluate(v any) bool {
	s, ok := convertStringValue(v)
	if !ok {
		return false
	}

	return strings.HasPrefix(s, e.ref)
}
