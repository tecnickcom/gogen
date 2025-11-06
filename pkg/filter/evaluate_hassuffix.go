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

// newHasSuffix returns an Evaluator that checks if a string ends with the reference suffix.
func newHasSuffix(r any) (Evaluator, error) {
	str, ok := r.(string)
	if !ok {
		return nil, fmt.Errorf("rule of type %s should have string value (got %v (%v))", TypeHasSuffix, r, reflect.TypeOf(r))
	}

	return &evalHasSuffix{ref: str}, nil
}

// Evaluate returns whether the input value ends with the reference string.
// It returns false if the input value is not a string.
func (e *evalHasSuffix) Evaluate(v any) bool {
	s, ok := convertStringValue(v)
	if !ok {
		return false
	}

	return strings.HasSuffix(s, e.ref)
}
