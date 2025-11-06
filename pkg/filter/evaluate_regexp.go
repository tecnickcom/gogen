package filter

import (
	"fmt"
	"reflect"
	"regexp"
)

// evalRegexp is an evaluator that checks if a string matches a regular expression.
type evalRegexp struct {
	rxp *regexp.Regexp
}

// newRegexp creates a new regexp evaluator from the given rule value.
func newRegexp(r any) (Evaluator, error) {
	str, ok := r.(string)
	if !ok {
		return nil, fmt.Errorf("rule of type %s should have string value (got %v (%v))", TypeRegexp, r, reflect.TypeOf(r))
	}

	reg, err := regexp.Compile(str)
	if err != nil {
		return nil, fmt.Errorf("failed compiling regexp: %w", err)
	}

	return &evalRegexp{rxp: reg}, nil
}

// Evaluate returns whether the input value matches the reference regular expression.
// It returns false if the input value is not a string.
func (e *evalRegexp) Evaluate(v any) bool {
	s, ok := convertStringValue(v)
	if !ok {
		return false
	}

	return e.rxp.MatchString(s)
}
