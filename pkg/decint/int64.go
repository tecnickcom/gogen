package decint

import (
	"fmt"
	"math"
	"strconv"
)

// FloatToInt converts a decimal float into the scaled int64 fixed-point form.
//
// The value is multiplied by 1e6 and truncated toward zero.
//
// Non-finite and out-of-range inputs are clamped (the signature is preserved):
// NaN yields 0, +Inf or values above MaxFloat yield MaxInt, and -Inf or values
// below -MaxFloat yield -MaxInt.
func FloatToInt(v float64) int64 {
	switch {
	case math.IsNaN(v):
		return 0
	case v >= MaxFloat:
		return MaxInt
	case v <= -MaxFloat:
		return -MaxInt
	}

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
//
// Unlike FloatToInt, which clamps non-finite and out-of-range values, this
// function returns an error when the parsed value is NaN, infinite, or outside
// the safe range [-MaxFloat, MaxFloat].
func StringToInt(s string) (int64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("unable to parse string number '%s': %w", s, err)
	}

	if math.IsNaN(v) || math.IsInf(v, 0) || v > MaxFloat || v < -MaxFloat {
		return 0, fmt.Errorf("number '%s' is not a finite value within the safe range [-%g, %g]", s, MaxFloat, MaxFloat)
	}

	return FloatToInt(v), nil
}

// IntToString formats a scaled int64 value as a six-decimal string.
//
// The fixed format provides stable output for serialization and comparisons.
func IntToString(v int64) string {
	return fmt.Sprintf(stringFormat, IntToFloat(v))
}
