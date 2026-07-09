package filter

import "reflect"

// not is an evaluator that negates the result of another evaluator.
type not struct {
	Not evaluator
}

// newNot constructs a negation evaluator that inverts the result of another evaluator.
func newNot(e evaluator) evaluator {
	return &not{Not: e}
}

// Evaluate returns the logical NOT of the inner evaluator's result.
func (n *not) Evaluate(v reflect.Value) bool {
	return !n.Not.Evaluate(v)
}
