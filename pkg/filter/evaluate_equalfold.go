package filter

import (
	"reflect"
	"strings"
)

// equalFold is an Evaluator that checks for equality under Unicode case-folding.
type equalFold struct {
	ref    any
	refNum numeric
	refOK  bool
}

// newEqualFold constructs a case-insensitive equality evaluator using Unicode case-folding.
// Numeric references are kept in an exact form to preserve large-integer precision.
func newEqualFold(r any) Evaluator {
	num, ok := toNumeric(r)

	return &equalFold{ref: convertValue(r), refNum: num, refOK: ok}
}

// Evaluate returns true for strings equal under Unicode case-folding (e.g., "AB" matches "ab"), with numeric normalization fallback.
// Two numeric operands are compared exactly to preserve large-integer precision.
func (e *equalFold) Evaluate(v any) bool {
	if e.refOK {
		if num, ok := toNumeric(v); ok {
			return e.refNum.equals(num)
		}

		return false
	}

	v = convertValue(v)

	val := reflect.ValueOf(v)
	ref := reflect.ValueOf(e.ref)

	if (val.Kind() == reflect.String) && (ref.Kind() == reflect.String) {
		return strings.EqualFold(val.String(), ref.String())
	}

	return (v == e.ref) || (isNil(v) && isNil(e.ref))
}
