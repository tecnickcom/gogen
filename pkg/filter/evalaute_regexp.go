package filter

import (
	"fmt"
	"reflect"
	"regexp"
)

type evalRegexp struct {
	internal *regexp.Regexp
}

func newRegexp(r interface{}) (Evaluator, error) {
	str, ok := r.(string)
	if !ok {
		return nil, fmt.Errorf("rule of type %s should have string value (got %v (%v))", TypeRegexp, r, reflect.TypeOf(r))
	}

	reg, err := regexp.Compile(str)
	if err != nil {
		return nil, fmt.Errorf("failed compiling regexp: %w", err)
	}

	return &evalRegexp{
		internal: reg,
	}, nil
}

// Evaluate returns whether the input value matches the reference regular expression.
// It returns false if the input value is not a string.
func (r *evalRegexp) Evaluate(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}

	return r.internal.MatchString(s)
}
