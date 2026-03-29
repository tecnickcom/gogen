package filter

import (
	"fmt"
	"reflect"
	"strings"
)

// evalContains is an Evaluator that checks if a string contains a reference substring.
type evalContains struct {
	ref string
}

// newContains constructs a substring-match evaluator from a reference string.
// Returns error if r is not a string.
func newContains(r any) (Evaluator, error) {
	str, ok := r.(string)
	if !ok {
		return nil, fmt.Errorf("rule of type %s should have string value (got %v (%v))", TypeContains, r, reflect.TypeOf(r))
	}

	return &evalContains{ref: str}, nil
}

// Evaluate returns true if the input string contains the reference substring, false if input is not a string.
func (e *evalContains) Evaluate(v any) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}

	return strings.Contains(s, e.ref)
}
