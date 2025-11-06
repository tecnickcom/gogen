package filter

// not is an evaluator that negates the result of another evaluator.
type not struct {
	Not Evaluator
}

// newNot creates a new not evaluator that negates the result of the given evaluator.
func newNot(e Evaluator) Evaluator {
	return &not{Not: e}
}

// Evaluate returns the opposite (boolean NOT) of the internal evaluator.
func (n *not) Evaluate(v any) bool {
	return !n.Not.Evaluate(v)
}
