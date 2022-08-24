package filter

import (
	"fmt"
	"reflect"
	"strings"
)

type evalHasPrefix struct {
	ref string
}

func newHasPrefix(r interface{}) (Evaluator, error) {
	str, ok := r.(string)
	if !ok {
		return nil, fmt.Errorf("rule of type %s should have string value (got %v (%v))", TypeHasPrefix, r, reflect.TypeOf(r))
	}

	return &evalHasPrefix{ref: str}, nil
}

// Evaluate returns whether the input value begins with the reference string.
// It returns false if the input value is not a string.
func (e *evalHasPrefix) Evaluate(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}

	return strings.HasPrefix(s, e.ref)
}
