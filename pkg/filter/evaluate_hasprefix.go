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

// newHasPrefix constructs a prefix-match evaluator from a reference string.
// Returns error if r is not a string.
func newHasPrefix(r any) (Evaluator, error) {
	str, ok := r.(string)
	if !ok {
		return nil, fmt.Errorf("rule of type %s should have string value (got %v (%v))", TypeHasPrefix, r, reflect.TypeOf(r))
	}

	return &evalHasPrefix{ref: str}, nil
}

// Evaluate returns true if the input string begins with the reference prefix, false if input is not a string.
func (e *evalHasPrefix) Evaluate(v any) bool {
	s, ok := convertStringValue(v)
	if !ok {
		return false
	}

	return strings.HasPrefix(s, e.ref)
}
