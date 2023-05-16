package filter

import (
	"fmt"
	"reflect"

	"github.com/Vonage/gosrvlib/pkg/typeutil"
)

// Evaluator is the interface to provide functions for a filter type.
type Evaluator interface {
	// Evaluate determines if two given values match.
	Evaluate(value any) bool
}

func isNil(v any) bool {
	return typeutil.IsNil(v)
}

//nolint:gocyclo
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

func convertStringValue(v any) (string, bool) {
	if v == nil {
		return "", false
	}

	if s, ok := v.(string); ok {
		return s, true
	}

	// Convert string aliases back to string
	vv := reflect.ValueOf(v)
	st := reflect.TypeOf("")

	if !vv.CanConvert(st) {
		return "", false
	}

	return vv.Convert(st).String(), true
}

func convertFloatValue(v any) (float64, error) {
	v = convertValue(v)

	if reflect.ValueOf(v).Kind() != reflect.Float64 {
		return 0, fmt.Errorf("rule value must be numerical (got %v (%v))", v, reflect.TypeOf(v))
	}

	return reflect.ValueOf(v).Float(), nil
}
