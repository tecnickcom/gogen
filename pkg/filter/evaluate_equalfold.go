package filter

import (
	"reflect"
	"strings"
)

// equalFold is an evaluator that checks for equality under Unicode case-folding.
type equalFold struct {
	ref    any
	refNum numeric
	refOK  bool
}

// newEqualFold constructs a case-insensitive equality evaluator using Unicode case-folding.
// Numeric references are kept in an exact form to preserve large-integer precision.
func newEqualFold(r any) evaluator {
	num, ok := toNumeric(r)

	return &equalFold{ref: convertValue(r), refNum: num, refOK: ok}
}

// Evaluate returns true for strings equal under Unicode case-folding (e.g., "AB" matches "ab"), with numeric normalization fallback.
// Two numeric operands are compared exactly to preserve large-integer precision.
// String and numeric operands are read from v without allocating; the deep-equal fallback
// boxes the field, and is reached for any reference that is neither numeric, nor a string,
// nor nil: a boolean, or an uncomparable dynamic type (a map or a slice).
func (e *equalFold) Evaluate(v reflect.Value) bool {
	if e.refOK {
		num, ok := toNumericValue(v)

		return ok && e.refNum.equals(num)
	}

	switch ref := e.ref.(type) {
	case string:
		s, ok := stringValue(v)

		return ok && strings.EqualFold(s, ref)
	case nil:
		return isNilValue(v)
	default:
		if !v.IsValid() {
			return false
		}

		return equalValues(convertValue(v.Interface()), ref)
	}
}
