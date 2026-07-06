package decint

import (
	"fmt"
	"math"
	"strconv"
)

// FloatToInt converts a decimal float into the scaled int64 fixed-point form.
//
// The value is multiplied by 1e6 and rounded to the nearest integer
// (half away from zero), so exact decimal inputs whose float64 form is one
// ULP off (e.g. 8.2) still map to their exact scaled value.
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

	return int64(math.Round(v * precision))
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
// function returns an error wrapping ErrInvalidNumber when the string cannot be
// parsed or is NaN or infinite, and ErrOutOfRange when the value is outside the
// safe range [-MaxFloat, MaxFloat].
func StringToInt(s string) (int64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: unable to parse '%s': %w", ErrInvalidNumber, s, err)
	}

	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("%w: '%s' is not finite", ErrInvalidNumber, s)
	}

	if v > MaxFloat || v < -MaxFloat {
		return 0, fmt.Errorf("%w: '%s' is outside [-%d, %d] scaled", ErrOutOfRange, s, int64(MaxInt), int64(MaxInt))
	}

	return FloatToInt(v), nil
}

// IntToString formats a scaled int64 value as a six-decimal string.
//
// The value is formatted directly from the integer, so the output is exact for
// every int64 (including values outside the safe range) and independent of
// float64 precision.
func IntToString(v int64) string {
	sign := ""

	u := uint64(v)
	if v < 0 {
		sign, u = "-", -u // modular negation yields the magnitude, even for math.MinInt64
	}

	return fmt.Sprintf("%s%d.%06d", sign, u/scale, u%scale)
}
