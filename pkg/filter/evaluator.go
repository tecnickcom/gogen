package filter

import (
	"fmt"
	"reflect"

	"github.com/tecnickcom/gogen/pkg/typeutil"
)

// Evaluator determines if a given value matches filter criteria.
type Evaluator interface {
	// Evaluate returns true when the value satisfies the evaluator condition.
	Evaluate(value any) bool
}

// isNil reports whether the given value is nil, handling typed-nil cases.
func isNil(v any) bool {
	return typeutil.IsNil(v)
}

// equalValues reports whether two normalized values are equal without panicking on
// non-comparable dynamic types (e.g. maps or slices decoded from untrusted JSON filters),
// which fall back to reflect.DeepEqual. Typed and untyped nils are considered equal.
func equalValues(a, b any) bool {
	if isNil(a) || isNil(b) {
		return isNil(a) && isNil(b)
	}

	if reflect.TypeOf(a).Comparable() && reflect.TypeOf(b).Comparable() {
		return a == b
	}

	return reflect.DeepEqual(a, b)
}

// convertValue normalizes numeric types (including named numeric types) to float64
// and string kinds to string for cross-type comparison, leaving others unchanged.
func convertValue(v any) any {
	if num, ok := toNumeric(v); ok {
		return num.float()
	}

	if s, ok := convertStringValue(v); ok {
		return s
	}

	return v
}

// convertStringValue attempts to coerce the given value to string via type assertion or string-alias conversion.
func convertStringValue(v any) (string, bool) {
	if v == nil {
		return "", false
	}

	if s, ok := v.(string); ok {
		return s, true
	}

	// Convert string aliases back to string.
	// Only string kinds qualify: CanConvert would also accept integer kinds
	// (rune-string conversion), which must not match string evaluators.
	vv := reflect.ValueOf(v)
	if vv.Kind() != reflect.String {
		return "", false
	}

	return vv.String(), true
}

// convertFloatValue normalizes the value to float64 using convertValue, returning error for non-numeric types.
func convertFloatValue(v any) (float64, error) {
	v = convertValue(v)

	if reflect.ValueOf(v).Kind() != reflect.Float64 {
		return 0, fmt.Errorf("rule value must be numerical (got %v (%v))", v, reflect.TypeOf(v))
	}

	return reflect.ValueOf(v).Float(), nil
}
