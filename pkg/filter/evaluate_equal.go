package filter

type equal struct {
	ref any
}

func newEqual(r any) Evaluator {
	return &equal{ref: convertValue(r)}
}

// Evaluate returns whether reference and actual value are considered equal.
// It converts numerical values implicitly before comparison.
func (e *equal) Evaluate(v any) bool {
	v = convertValue(v)

	return (v == e.ref) || (isNil(v) && isNil(e.ref))
}
