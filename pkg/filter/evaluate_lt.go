package filter

// lt is an Evaluator that checks if a value is less than a reference.
type lt struct {
	order
}

// newLT constructs a less-than evaluator from a reference numeric value.
// Returns error if r cannot be converted to a numeric value.
func newLT(r any) (Evaluator, error) {
	o, err := newOrder(r)
	if err != nil {
		return nil, err
	}

	return &lt{order: o}, nil
}

// Evaluate returns true if the numeric value is less than reference, or collection length is less than reference for arrays/maps/slices/strings.
func (e *lt) Evaluate(v any) bool {
	c, ok := e.compare(v)

	return ok && c < 0
}
