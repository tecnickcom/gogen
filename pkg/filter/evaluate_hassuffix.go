package filter

import (
	"fmt"
	"reflect"
	"strings"
)

// evalHasSuffix is an Evaluator that checks if a string ends with a reference suffix.
type evalHasSuffix struct {
	ref string
}

// newHasSuffix constructs a suffix-match evaluator from a reference string.
// Returns error if r is not a string.
func newHasSuffix(r any) (Evaluator, error) {
	str, ok := r.(string)
	if !ok {
		return nil, fmt.Errorf("rule of type %s should have string value (got %v (%v))", TypeHasSuffix, r, reflect.TypeOf(r))
	}

	return &evalHasSuffix{ref: str}, nil
}

// Evaluate returns true if the input string ends with the reference suffix, false if input is not a string.
func (e *evalHasSuffix) Evaluate(v any) bool {
	s, ok := convertStringValue(v)
	if !ok {
		return false
	}

	return strings.HasSuffix(s, e.ref)
}
