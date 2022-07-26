package filter

import (
	"fmt"
	"reflect"
	"regexp"
)

const (
	// TypeEqual is a filter type that matches exactly the reference value.
	TypeEqual = "equal"

	// TypeNotEqual is a filter type that matches when the value is different from the reference value (opposite of TypeEqual).
	TypeNotEqual = "notequal"

	// TypeRegexp is a filter type that matches the value against a reference regular expression.
	// The reference value must be a regular expression that can compile.
	// Works only with strings (anything else will evaluate to false).
	TypeRegexp = "regexp"
)

// Evaluator is the interface to provide functions for a filter type.
type Evaluator interface {
	// Evaluate determines if two given values match.
	Evaluate(value interface{}) bool
}

type equal struct {
	ref interface{}
}

func newEqual(reference interface{}) Evaluator {
	return &equal{
		ref: convertValues(reference),
	}
}

// Evaluate returns whether reference and actual are considered equal.
// It converts numerical values implicitly before comparison.
func (e *equal) Evaluate(value interface{}) bool {
	value = convertValues(value)

	if value == e.ref {
		return true
	}

	if e.isNil(value) && e.isNil(e.ref) {
		return true
	}

	return false
}

func (e *equal) isNil(v interface{}) bool {
	if v == nil {
		return true
	}

	value := reflect.ValueOf(v)
	if (value.Kind() == reflect.Interface || value.Kind() == reflect.Ptr) && value.IsNil() {
		return true
	}

	return false
}

// nolint: gocyclo
func convertValues(value interface{}) interface{} {
	switch value := value.(type) {
	case int:
		return float64(value)
	case int8:
		return float64(value)
	case int16:
		return float64(value)
	case int32:
		return float64(value)
	case int64:
		return float64(value)
	case uint:
		return float64(value)
	case uint8:
		return float64(value)
	case uint16:
		return float64(value)
	case uint32:
		return float64(value)
	case uint64:
		return float64(value)
	case float32:
		return float64(value)
	default:
		return value
	}
}

type not struct {
	Opposite Evaluator
}

func newNot(opposite Evaluator) Evaluator {
	return &not{
		Opposite: opposite,
	}
}

// Evaluate returns the opposite of the internal evaluator.
func (n *not) Evaluate(value interface{}) bool {
	return !n.Opposite.Evaluate(value)
}

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
