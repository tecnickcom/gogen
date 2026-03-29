package filter

import (
	"reflect"
	"strings"
)

// equalFold is an Evaluator that checks for equality under Unicode case-folding.
type equalFold struct {
	ref any
}

// newEqualFold constructs a case-insensitive equality evaluator using Unicode case-folding.
func newEqualFold(r any) Evaluator {
	return &equalFold{ref: convertValue(r)}
}

// Evaluate returns true for strings equal under Unicode case-folding (e.g., "AB" matches "ab"), with numeric normalization fallback.
func (e *equalFold) Evaluate(v any) bool {
	v = convertValue(v)

	val := reflect.ValueOf(v)
	ref := reflect.ValueOf(e.ref)

	if (val.Kind() == reflect.String) && (ref.Kind() == reflect.String) {
		return strings.EqualFold(val.String(), ref.String())
	}

	return (v == e.ref) || (isNil(v) && isNil(e.ref))
}
