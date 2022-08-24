package filter

import (
	"fmt"
	"reflect"
	"strings"
)

type evalContains struct {
	ref string
}

func newContains(r interface{}) (Evaluator, error) {
	str, ok := r.(string)
	if !ok {
		return nil, fmt.Errorf("rule of type %s should have string value (got %v (%v))", TypeContains, r, reflect.TypeOf(r))
	}

	return &evalContains{ref: str}, nil
}

// Evaluate returns whether the input value contains the reference string.
// It returns false if the input value is not a string.
func (e *evalContains) Evaluate(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}

	return strings.Contains(s, e.ref)
}
