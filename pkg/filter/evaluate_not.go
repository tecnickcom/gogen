package filter

// not is an evaluator that negates the result of another evaluator.
type not struct {
	Not Evaluator
}

// newNot constructs a negation evaluator that inverts the result of another evaluator.
func newNot(e Evaluator) Evaluator {
	return &not{Not: e}
}

// Evaluate returns the logical NOT of the inner evaluator's result.
func (n *not) Evaluate(v any) bool {
	return !n.Not.Evaluate(v)
}
