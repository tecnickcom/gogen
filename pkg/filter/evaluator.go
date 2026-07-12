package filter

import (
	"fmt"
	"reflect"

	"github.com/tecnickcom/nurago/pkg/typeutil"
)

// evaluator determines if a given value matches filter criteria.
type evaluator interface {
	// Evaluate returns true when the value satisfies the evaluator condition.
	//
	// The value is passed as a [reflect.Value] so that filtering large slices does not box
	// every element and field into an any: elements are never boxed, and string and numeric
	// operands are read directly from the reflect.Value without allocating. A field is boxed
	// only by the deep-equal fallback of the equality evaluators, which is reached when the
	// rule's reference value is neither numeric, nor a string, nor nil (a boolean, or an
	// uncomparable map or slice value). An invalid Value (the zero [reflect.Value])
	// represents a nil or absent operand.
	Evaluate(v reflect.Value) bool
}

// isNil reports whether the given value is nil, handling typed-nil cases.
func isNil(v any) bool {
	return typeutil.IsNil(v)
}

// isNilValue reports whether v represents a nil operand: an invalid Value (an
// absent field or a nil interface) or a typed nil (pointer, slice, map, channel,
// function). It is the [reflect.Value] counterpart of [isNil].
func isNilValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}

	switch v.Kind() { //nolint:exhaustive
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return v.IsNil()
	}

	return false
}

// stringValue extracts the underlying string of a string-kinded value (including
// named string types such as "type ID string") without allocating. ok is false
// for any other kind, or for an invalid Value.
func stringValue(v reflect.Value) (string, bool) {
	if v.IsValid() && v.Kind() == reflect.String {
		return v.String(), true
	}

	return "", false
}

// toNumericValue normalizes a numeric-kinded [reflect.Value] into an exact numeric,
// reading the value directly to avoid boxing it into an any. ok is false for
// non-numeric kinds or an invalid Value. It is the reflect.Value counterpart of
// [toNumericReflect] and preserves large-integer exactness the same way.
func toNumericValue(v reflect.Value) (numeric, bool) {
	if !v.IsValid() {
		return numeric{}, false
	}

	//nolint:exhaustive
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return numeric{kind: numericInt, i: v.Int()}, true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return numeric{kind: numericUint, u: v.Uint()}, true
	case reflect.Float32, reflect.Float64:
		return numeric{kind: numericFloat, f: v.Float()}, true
	}

	return numeric{}, false
}

// equalValues reports whether two normalized values are equal without panicking on
// non-comparable dynamic types (e.g. maps or slices held by an any field), which fall back to
// reflect.DeepEqual. Typed and untyped nils are considered equal.
func equalValues(a, b any) bool {
	if isNil(a) || isNil(b) {
		return isNil(a) && isNil(b)
	}

	if reflect.TypeOf(a).Comparable() && reflect.TypeOf(b).Comparable() {
		return comparableEqual(a, b)
	}

	return reflect.DeepEqual(a, b)
}

// comparableEqual reports a == b for operands whose types report Comparable, deferring to
// reflect.DeepEqual if the comparison panics. Type.Comparable is a property of the static
// type: a struct or array whose field/element type is an interface reports true, yet == still
// panics at runtime when that interface holds an uncomparable value (a slice or a map). This
// is only reachable from a Go-constructed rule; JSON rule values are scalars.
func comparableEqual(a, b any) bool {
	equal := false

	func() {
		defer func() {
			if recover() != nil {
				equal = reflect.DeepEqual(a, b)
			}
		}()

		equal = a == b
	}()

	return equal
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
		return 0, fmt.Errorf("%w: rule value must be numerical (got %v (%v))", ErrInvalidFilter, v, reflect.TypeOf(v))
	}

	return reflect.ValueOf(v).Float(), nil
}
