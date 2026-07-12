package filter

import "reflect"

// equal evaluates exact equality against a reference value.
type equal struct {
	ref    any
	refNum numeric
	refOK  bool
}

// newEqual constructs an equality evaluator from a reference value.
// Numeric references are kept in an exact form so that large int64/uint64 values
// compare without the precision loss of widening to float64.
func newEqual(r any) evaluator {
	num, ok := toNumeric(r)

	return &equal{ref: convertValue(r), refNum: num, refOK: ok}
}

// Evaluate returns true if reference and value are equal (with numeric normalization) or both nil.
// Two numeric operands are compared exactly to preserve large-integer precision.
// String and numeric operands are read from v without allocating; the deep-equal fallback
// boxes the field, and is reached for any reference that is neither numeric, nor a string,
// nor nil: a boolean, or an uncomparable dynamic type (a map or a slice).
func (e *equal) Evaluate(v reflect.Value) bool {
	if e.refOK {
		num, ok := toNumericValue(v)

		return ok && e.refNum.equals(num)
	}

	switch ref := e.ref.(type) {
	case string:
		s, ok := stringValue(v)

		return ok && s == ref
	case nil:
		return isNilValue(v)
	default:
		if !v.IsValid() {
			return false
		}

		return equalValues(convertValue(v.Interface()), ref)
	}
}
