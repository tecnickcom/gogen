package filter

import (
	"reflect"
	"strings"
)

type equalFold struct {
	ref interface{}
}

func newEqualFold(r interface{}) Evaluator {
	return &equalFold{ref: convertValue(r)}
}

// Evaluate returns whether reference and actual value are considered equal under simple Unicode case-folding, which is a more general form of case-insensitivity.
// For example "AB" will match "ab".
// It converts numerical values implicitly before comparison.
func (e *equalFold) Evaluate(v interface{}) bool {
	v = convertValue(v)

	val := reflect.ValueOf(v)
	ref := reflect.ValueOf(e.ref)

	if (val.Kind() == reflect.String) && (ref.Kind() == reflect.String) {
		return strings.EqualFold(val.String(), ref.String())
	}

	return (v == e.ref) || (isNil(v) && isNil(e.ref))
}
