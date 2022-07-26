package filter

import (
	"fmt"
	"strings"
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
	switch strings.ToLower(r.Type) {
	case TypeEqual:
		return newEqual(r.Value), nil
	case TypeNotEqual:
		return newNot(newEqual(r.Value)), nil
	case TypeRegexp:
		return newRegexp(r.Value)
	default:
		return nil, fmt.Errorf("type %s is not supported", r.Type)
	}
}
