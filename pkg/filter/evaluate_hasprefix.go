package filter

import (
	"fmt"
	"reflect"
	"strings"
)

// evalHasPrefix is an evaluator that checks if a string begins with a reference prefix.
type evalHasPrefix struct {
	ref string
}

// newHasPrefix constructs a prefix-match evaluator from a reference string.
// Returns error if r is not a string.
func newHasPrefix(r any) (evaluator, error) {
	str, ok := r.(string)
	if !ok {
		return nil, fmt.Errorf("%w: rule of type %s should have string value (got %v (%v))", ErrInvalidFilter, TypeHasPrefix, r, reflect.TypeOf(r))
	}

	return &evalHasPrefix{ref: str}, nil
}

// Evaluate returns true if the input string begins with the reference prefix, false if input is not a string.
func (e *evalHasPrefix) Evaluate(v reflect.Value) bool {
	s, ok := stringValue(v)
	if !ok {
		return false
	}

	return strings.HasPrefix(s, e.ref)
}
