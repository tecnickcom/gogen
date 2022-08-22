package filter

import (
	"fmt"
	"reflect"
	"regexp"
)

const (
	// TypeRegexp is a filter type that matches the value against a reference regular expression.
	// The reference value must be a regular expression that can compile.
	// Works only with strings (anything else will evaluate to false).
	TypeRegexp = "regexp"
)

type evalRegexp struct {
	internal *regexp.Regexp
}

func newRegexp(reference interface{}) (Evaluator, error) {
	str, ok := reference.(string)
	if !ok {
		return nil, fmt.Errorf("rule of type %s should have string values (got %v (%v))", TypeRegexp, reference, reflect.TypeOf(reference))
	}

	reg, err := regexp.Compile(str)
	if err != nil {
		return nil, fmt.Errorf("failed compiling regexp: %w", err)
	}

	return &evalRegexp{
		internal: reg,
	}, nil
}

// Evaluate returns whether actual matches the reference regexp.
// It returns an error if reference is not a string or a valid regular expression.
// It returns false if actual is not a string.
func (r *evalRegexp) Evaluate(value interface{}) bool {
	actualStr, ok := value.(string)
	if !ok {
		return false
	}

	return r.internal.MatchString(actualStr)
}
