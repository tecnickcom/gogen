package decint

import (
	"fmt"
	"math"
	"strconv"
)

// FloatToUint converts a decimal float into the scaled uint64 fixed-point form.
//
// The value is multiplied by 1e6 and rounded to the nearest integer, so exact
// decimal inputs whose float64 form is one ULP off (e.g. 8.2) still map to
// their exact scaled value.
//
// Values less than or equal to zero are clamped to 0, making this helper safe
// for unsigned amount domains.
//
// Non-finite and out-of-range inputs are clamped (the signature is preserved):
// NaN and values less than or equal to zero (including -Inf) yield 0, while
// +Inf or values above MaxFloat yield MaxInt.
func FloatToUint(v float64) uint64 {
	switch {
	case math.IsNaN(v):
		return 0
	case v <= 0:
		return 0
	case v >= MaxFloat:
		return MaxInt
	}

	return uint64(math.Round(v * precision))
}

// UintToFloat converts a scaled uint64 fixed-point value back to float64.
func UintToFloat(v uint64) float64 {
	return float64(v) / precision
}

// StringToUint parses a decimal string and returns its scaled uint64 form.
//
// Parsed values less than or equal to zero are clamped to 0.
//
// Unlike FloatToUint, which clamps non-finite and out-of-range values, this
// function returns an error when the parsed value is NaN, infinite, or above
// the safe maximum MaxFloat.
func StringToUint(s string) (uint64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("unable to parse string number '%s': %w", s, err)
	}

	if math.IsNaN(v) || math.IsInf(v, 0) || v > MaxFloat {
		return 0, fmt.Errorf("number '%s' is not a finite value within the safe range [0, %g]", s, MaxFloat)
	}

	return FloatToUint(v), nil
}

// UintToString formats a scaled uint64 value as a six-decimal string.
//
// The fixed format keeps textual output deterministic across callers.
func UintToString(v uint64) string {
	return fmt.Sprintf(stringFormat, UintToFloat(v))
}
