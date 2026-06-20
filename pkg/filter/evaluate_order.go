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
func (o order) compare(v any) (int, bool) {
	if isNil(v) {
		return 0, false
	}

	if num, ok := toNumeric(v); ok {
		return num.compare(o.ref)
	}

	v = convertValue(v)

	val := reflect.ValueOf(v)

	//nolint:exhaustive
	switch val.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		length := numeric{kind: numericInt, i: int64(val.Len())}

		return length.compare(o.ref)
	}

	return 0, false
}
