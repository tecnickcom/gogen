package decint

import (
	"fmt"
	"strconv"
)

// FloatToInt Converts the value to a int64 representation.
func FloatToInt(v float64) int64 {
	return int64(v * precision)
}

// IntToFloat Converts back the int64 representation into a float value.
func IntToFloat(v int64) float64 {
	return float64(v) / precision
}

// StringToInt parse the input string float value and returns the int64 representation.
func StringToInt(s string) (int64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("unable to parse string number '%s': %w", s, err)
	}

	return FloatToInt(v), nil
}

// IntToString format the int64 representation as string.
func IntToString(v int64) string {
	return fmt.Sprintf(stringFormat, IntToFloat(v))
}
