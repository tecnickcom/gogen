package filter

// equal is an Evaluator that checks for equality.
type equal struct {
	ref any
}

// newEqual returns an Evaluator that checks for equality.
func newEqual(r any) Evaluator {
	return &equal{ref: convertValue(r)}
}

// Evaluate returns whether reference and actual value are considered equal.
// It converts numerical values implicitly before comparison.
func (e *equal) Evaluate(v any) bool {
	v = convertValue(v)

	return (v == e.ref) || (isNil(v) && isNil(e.ref))
}
