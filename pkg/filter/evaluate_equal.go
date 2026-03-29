package filter

// equal evaluates exact equality against a reference value.
type equal struct {
	ref any
}

// newEqual constructs an equality evaluator from a reference value, normalizing numeric types to float64.
func newEqual(r any) Evaluator {
	return &equal{ref: convertValue(r)}
}

// Evaluate returns true if reference and value are equal (with numeric normalization) or both nil.
func (e *equal) Evaluate(v any) bool {
	v = convertValue(v)

	return (v == e.ref) || (isNil(v) && isNil(e.ref))
}
