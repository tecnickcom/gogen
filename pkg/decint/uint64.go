package decint

import (
	"fmt"
	"strconv"
)

// FloatToUint Converts the value to a uint64 representation.
// Negative values are converted to zero.
func FloatToUint(v float64) uint64 {
	if v <= 0 {
		return 0
	}

	return uint64(v * precision)
}

// UintToFloat Converts back the uint64 representation into a float value.
func UintToFloat(v uint64) float64 {
	return float64(v) / precision
}

// StringToUint parse the input string float value and returns the uint64 representation.
// Negative values are converted to zero.
func StringToUint(s string) (uint64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("unable to parse string number '%s': %w", s, err)
	}

	return FloatToUint(v), nil
}

// UintToString format the uint64 representation as string.
func UintToString(v uint64) string {
	return fmt.Sprintf(stringFormat, UintToFloat(v))
}
