package decint

import (
	"fmt"
	"math"
	"strconv"
)

// FloatToUint converts a decimal float into the scaled uint64 fixed-point form.
//
// The value is multiplied by 1e6 and rounded to the nearest integer (half away
// from zero), so exact decimal inputs whose float64 form is one ULP off
// (e.g. 8.2) still map to their exact scaled value.
//
// Values less than or equal to zero are clamped to 0.
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
// The rules mirror the unsigned domain:
//   - a string that cannot be parsed, or that is NaN or infinite (either sign),
//     returns an error wrapping ErrInvalidNumber;
//   - a finite value less than or equal to zero is clamped to 0 (no error);
//   - a finite value above MaxFloat returns an error wrapping ErrOutOfRange.
func StringToUint(s string) (uint64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: unable to parse '%s': %w", ErrInvalidNumber, s, err)
	}

	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("%w: '%s' is not finite", ErrInvalidNumber, s)
	}

	if v > MaxFloat {
		return 0, fmt.Errorf("%w: '%s' is above %d scaled", ErrOutOfRange, s, int64(MaxInt))
	}

	return FloatToUint(v), nil
}

// UintToString formats a scaled uint64 value as a six-decimal string.
//
// The value is formatted directly from the integer, so the output is exact for
// every uint64 (including values outside the safe range) and independent of
// float64 precision.
func UintToString(v uint64) string {
	return fmt.Sprintf("%d.%06d", v/scale, v%scale)
}
