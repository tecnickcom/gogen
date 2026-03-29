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

// convertValue normalizes numeric types to float64 for cross-type comparison, leaving others unchanged.
//
//nolint:gocyclo,cyclop
func convertValue(v any) any {
	switch v := v.(type) {
	case int:
		return float64(v)
	case int8:
		return float64(v)
	case int16:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case uint:
		return float64(v)
	case uint8:
		return float64(v)
	case uint16:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	case float32:
		return float64(v)
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

	// Convert string aliases back to string
	vv := reflect.ValueOf(v)
	st := reflect.TypeFor[string]()

	if !vv.CanConvert(st) {
		return "", false
	}

	return vv.Convert(st).String(), true
}

// convertFloatValue normalizes the value to float64 using convertValue, returning error for non-numeric types.
func convertFloatValue(v any) (float64, error) {
	v = convertValue(v)

	if reflect.ValueOf(v).Kind() != reflect.Float64 {
		return 0, fmt.Errorf("rule value must be numerical (got %v (%v))", v, reflect.TypeOf(v))
	}

	return reflect.ValueOf(v).Float(), nil
}
