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

// newRegexp constructs a regex-match evaluator from a reference regex pattern.
// Returns error if r is not a string or if the pattern fails to compile.
func newRegexp(r any) (evaluator, error) {
	str, ok := r.(string)
	if !ok {
		return nil, fmt.Errorf("%w: rule of type %s should have string value (got %v (%v))", ErrInvalidFilter, TypeRegexp, r, reflect.TypeOf(r))
	}

	reg, err := regexp.Compile(str)
	if err != nil {
		return nil, fmt.Errorf("%w: failed compiling regexp: %w", ErrInvalidFilter, err)
	}

	return &evalRegexp{rxp: reg}, nil
}

// Evaluate returns true if the input string matches the reference regex, false if input is not a string.
func (e *evalRegexp) Evaluate(v reflect.Value) bool {
	s, ok := stringValue(v)
	if !ok {
		return false
	}

	return e.rxp.MatchString(s)
}
