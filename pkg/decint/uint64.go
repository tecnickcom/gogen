package decint

import (
	"fmt"
	"strconv"
)

// FloatToUint converts a decimal float into the scaled uint64 fixed-point form.
//
// Values less than or equal to zero are clamped to 0, making this helper safe
// for unsigned amount domains.
func FloatToUint(v float64) uint64 {
	if v <= 0 {
		return 0
	}

	return uint64(v * precision)
}

// UintToFloat converts a scaled uint64 fixed-point value back to float64.
func UintToFloat(v uint64) float64 {
	return float64(v) / precision
}

// StringToUint parses a decimal string and returns its scaled uint64 form.
//
// Parsed values less than or equal to zero are clamped to 0.
func StringToUint(s string) (uint64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("unable to parse string number '%s': %w", s, err)
	}

	return FloatToUint(v), nil
}

// UintToString formats a scaled uint64 value as a six-decimal string.
//
// The fixed format keeps textual output deterministic across callers.
func UintToString(v uint64) string {
	return fmt.Sprintf(stringFormat, UintToFloat(v))
}
