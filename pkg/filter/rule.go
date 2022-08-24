package filter

import (
	"fmt"
	"strings"
)

const (
	// TypePrefixNot is a prefix that can be added to any type to get the negated value (opposite match).
	TypePrefixNot = "!"

	// TypeRegexp is a filter type that matches the value against a reference regular expression.
	// The reference value must be a regular expression that can compile.
	// Works only with strings (anything else will evaluate to false).
	TypeRegexp = "regexp"

	// TypeEqual is a filter type that matches exactly the reference value.
	TypeEqual = "=="

	// TypeEqualFold is a filter type that matches when strings, interpreted as UTF-8, are equal under simple Unicode case-folding, which is a more general form of case-insensitivity. For example "AB" will match "ab".
	TypeEqualFold = "="

	// TypeHasPrefix is a filter type that matches when the value begins with the reference string.
	TypeHasPrefix = "^="

	// TypeHasSuffix  is a filter type that matches when the value ends with the reference string.
	TypeHasSuffix = "=$"

	// TypeContains  is a filter type that matches when the reference string is a sub-string of the value.
	TypeContains = "~="

	// TypeLT is a filter type that matches when the value is less than reference.
	TypeLT = "<"

	// TypeLTE is a filter type that matches when the value is less than or equal the reference.
	TypeLTE = "<="

	// TypeGT is a filter type that matches when the value is greater than reference.
	TypeGT = ">"

	// TypeGTE is a filter type that matches when the value is greater than or equal the reference.
	TypeGTE = ">="
)

// Rule is an individual filter that can be evaluated against any value.
type Rule struct {
	// Field is a dot separated selector that is used to target a specific field of the evaluated value.
	//
	// * "Age" will select the Age field of a structure
	// * "Address.Country" will select the Country subfield of the Address structure
	// * "" will select the whole value (e.g. to filter a []string)
	Field string `json:"field"`

	// Type controls the evaluation to apply.
	// An invalid value will cause Evaluate() to return an error.
	// See the Type* constants of this package for valid values.
	Type string `json:"type"`

	// Value is the reference value to evaluate against.
	// Its type should be accepted by the chosen Type.
	Value interface{} `json:"value"`

	// eval is initialized at the first call to Evaluate() and stores the structure that evaluates the rule.
	eval Evaluator
}

// Evaluate returns whether the value matches the rule or not.
//
// Returns an error if the Type is invalid, a misconfiguration (e.g. invalid regexp) or the value is invalid (e.g. evaluating an int with a regexp).
func (r *Rule) Evaluate(value interface{}) (bool, error) {
	if r.eval == nil {
		var err error

		r.eval, err = r.getEvaluator()
		if err != nil {
			return false, err
		}
	}

	return r.eval.Evaluate(value), nil
}

func (r *Rule) getEvaluator() (Evaluator, error) {
	t := strings.ToLower(r.Type)

	if strings.HasPrefix(t, TypePrefixNot) {
		e, err := r.getBaseTypeEvaluator(strings.TrimPrefix(t, TypePrefixNot))
		if err != nil {
			return nil, err
		}

		return newNot(e), nil
	}

	return r.getBaseTypeEvaluator(t)
}

//nolint:gocyclo
func (r *Rule) getBaseTypeEvaluator(t string) (Evaluator, error) {
	switch t {
	case TypeRegexp:
		return newRegexp(r.Value)
	case TypeEqual:
		return newEqual(r.Value), nil
	case TypeEqualFold:
		return newEqualFold(r.Value), nil
	case TypeHasPrefix:
		return newHasPrefix(r.Value)
	case TypeHasSuffix:
		return newHasSuffix(r.Value)
	case TypeContains:
		return newContains(r.Value)
	case TypeLT:
		return newLT(r.Value)
	case TypeLTE:
		return newLTE(r.Value)
	case TypeGT:
		return newGT(r.Value)
	case TypeGTE:
		return newGTE(r.Value)
	default:
		return nil, fmt.Errorf("type %s is not supported", r.Type)
	}
}
