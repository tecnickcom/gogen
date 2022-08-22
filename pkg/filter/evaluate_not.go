package filter

const (
	// TypeNotEqual is a filter type that matches when the value is different from the reference value (opposite of TypeEqual).
	TypeNotEqual = "notequal"
)

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
