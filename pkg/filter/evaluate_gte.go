package filter

// gte is an Evaluator that checks if a value is greater than or equal to a reference.
type gte struct {
	order
}

// newGTE constructs a greater-than-or-equal evaluator from a reference numeric value.
// Returns error if r cannot be converted to a numeric value.
func newGTE(r any) (Evaluator, error) {
	o, err := newOrder(r)
	if err != nil {
		return nil, err
	}

	return &gte{order: o}, nil
}

// Evaluate returns true if the numeric value is >= reference, or collection length is >= reference for arrays/maps/slices/strings.
func (e *gte) Evaluate(v any) bool {
	c, ok := e.compare(v)

	return ok && c >= 0
}
