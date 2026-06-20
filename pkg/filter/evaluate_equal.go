package filter

// equal evaluates exact equality against a reference value.
type equal struct {
	ref    any
	refNum numeric
	refOK  bool
}

// newEqual constructs an equality evaluator from a reference value.
// Numeric references are kept in an exact form so that large int64/uint64 values
// compare without the precision loss of widening to float64.
func newEqual(r any) Evaluator {
	num, ok := toNumeric(r)

	return &equal{ref: convertValue(r), refNum: num, refOK: ok}
}

// Evaluate returns true if reference and value are equal (with numeric normalization) or both nil.
// Two numeric operands are compared exactly to preserve large-integer precision.
func (e *equal) Evaluate(v any) bool {
	if e.refOK {
		if num, ok := toNumeric(v); ok {
			return e.refNum.equals(num)
		}

		return false
	}

	v = convertValue(v)

	return (v == e.ref) || (isNil(v) && isNil(e.ref))
}
