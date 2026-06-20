package filter

// lte is an Evaluator that checks if a value is less than or equal to a reference.
type lte struct {
	order
}

// newLTE constructs a less-than-or-equal evaluator from a reference numeric value.
// Returns error if r cannot be converted to a numeric value.
func newLTE(r any) (Evaluator, error) {
	o, err := newOrder(r)
	if err != nil {
		return nil, err
	}

	return &lte{order: o}, nil
}

// Evaluate returns true if the numeric value is <= reference, or collection length is <= reference for arrays/maps/slices/strings.
func (e *lte) Evaluate(v any) bool {
	c, ok := e.compare(v)

	return ok && c <= 0
}
