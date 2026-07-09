package filter

import "reflect"

// gt evaluates whether a value is greater than a reference.
type gt struct {
	order
}

// newGT constructs a greater-than evaluator from a reference numeric value.
// Returns error if r cannot be converted to a numeric value.
func newGT(r any) (evaluator, error) {
	o, err := newOrder(r)
	if err != nil {
		return nil, err
	}

	return &gt{order: o}, nil
}

// Evaluate returns true if the numeric value exceeds the reference, or collection length exceeds reference for arrays/maps/slices/strings.
func (e *gt) Evaluate(v reflect.Value) bool {
	c, ok := e.compare(v)

	return ok && c > 0
}
