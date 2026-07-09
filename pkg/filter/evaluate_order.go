package filter

import "reflect"

// order holds the reference for the numeric/length ordering evaluators (<, <=, >, >=).
// The reference is always numeric (constructors reject non-numeric values).
type order struct {
	ref numeric
}

// newOrder builds the shared ordering reference, returning an error for non-numeric references.
func newOrder(r any) (order, error) {
	// Validate that the reference is numeric (preserves the existing error message).
	_, err := convertFloatValue(r)
	if err != nil {
		return order{}, err
	}

	// A reference accepted by convertFloatValue is always one of the numeric types,
	// so toNumeric cannot fail here; keep its exact (non-widened) form.
	num, _ := toNumeric(r)

	return order{ref: num}, nil
}

// compare resolves the value against the reference and reports the comparison sign.
// It compares numbers exactly (preserving large-integer precision) and falls back to
// collection length for arrays, maps, slices and strings. ok is false when no ordering applies.
// The value is read from v without allocating.
func (o order) compare(v reflect.Value) (int, bool) {
	if isNilValue(v) {
		return 0, false
	}

	if num, ok := toNumericValue(v); ok {
		return num.compare(o.ref)
	}

	//nolint:exhaustive
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		length := numeric{kind: numericInt, i: int64(v.Len())}

		return length.compare(o.ref)
	}

	return 0, false
}
