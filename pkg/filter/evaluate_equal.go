package filter

type equal struct {
	ref interface{}
}

func newEqual(r interface{}) Evaluator {
	return &equal{ref: convertValue(r)}
}

// Evaluate returns whether reference and actual value are considered equal.
// It converts numerical values implicitly before comparison.
func (e *equal) Evaluate(v interface{}) bool {
	v = convertValue(v)

	return (v == e.ref) || (isNil(v) && isNil(e.ref))
}
