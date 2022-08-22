package filter

import (
	"reflect"
)

// Evaluator is the interface to provide functions for a filter type.
type Evaluator interface {
	// Evaluate determines if two given values match.
	Evaluate(value interface{}) bool
}

func isNil(v interface{}) bool {
	if v == nil {
		return true
	}

	value := reflect.ValueOf(v)
	if (value.Kind() == reflect.Interface || value.Kind() == reflect.Ptr) && value.IsNil() {
		return true
	}

	return false
}

// nolint: gocyclo
func convertValues(value interface{}) interface{} {
	switch value := value.(type) {
	case int:
		return float64(value)
	case int8:
		return float64(value)
	case int16:
		return float64(value)
	case int32:
		return float64(value)
	case int64:
		return float64(value)
	case uint:
		return float64(value)
	case uint8:
		return float64(value)
	case uint16:
		return float64(value)
	case uint32:
		return float64(value)
	case uint64:
		return float64(value)
	case float32:
		return float64(value)
	default:
		return value
	}
}
