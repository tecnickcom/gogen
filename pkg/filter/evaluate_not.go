package filter

type not struct {
	Not Evaluator
}

func newNot(e Evaluator) Evaluator {
	return &not{Not: e}
}

// Evaluate returns the opposite (boolean NOT) of the internal evaluator.
func (n *not) Evaluate(v any) bool {
	return !n.Not.Evaluate(v)
}
