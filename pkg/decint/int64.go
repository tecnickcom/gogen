package decint

import (
	"fmt"
	"strconv"
)

// FloatToInt converts a decimal float into the scaled int64 fixed-point form.
//
// The value is multiplied by 1e6 and truncated toward zero.
func FloatToInt(v float64) int64 {
	return int64(v * precision)
}

// IntToFloat converts a scaled int64 fixed-point value back to float64.
func IntToFloat(v int64) float64 {
	return float64(v) / precision
}

// StringToInt parses a decimal string and returns its scaled int64 form.
//
// This is useful when ingesting textual values from config, APIs, or storage
// while preserving the package's fixed-point representation.
func StringToInt(s string) (int64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("unable to parse string number '%s': %w", s, err)
	}

	return FloatToInt(v), nil
}

// IntToString formats a scaled int64 value as a six-decimal string.
//
// The fixed format provides stable output for serialization and comparisons.
func IntToString(v int64) string {
	return fmt.Sprintf(stringFormat, IntToFloat(v))
}
