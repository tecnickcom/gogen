/*
Package decint provides utility functions to parse and represent decimal values
as fixed-point integers with a defined precision.

This package solves the common problem of safely handling small monetary and
fixed-precision decimal values without using floating-point arithmetic for
storage or comparison.

Decint is designed for values with at most six decimal places, and it stores
those values as integers (scaled by 1e6) to preserve deterministic behavior in
comparisons, serialization, and transport.

Top features:

- bidirectional conversion between float64/string and scaled int64 or uint64
- fixed six-decimal output formatting for stable text representation
- explicit parse errors for invalid numeric strings
- published safe range constants (MaxInt and MaxFloat) for boundary checks
- unsigned conversion helpers that clamp non-positive values to zero

Implementation note:

  - float-to-integer conversion scales by 1e6 and rounds to the nearest
    integer (half away from zero), so extra fractional digits beyond the
    supported precision are rounded rather than truncated.

Key benefits:

- deterministic decimal handling for currency, rates, and small amounts
- exact six-decimal representation for every value within the safe range

Safe range:

  - Values are safe up to MaxFloat = 2^33 = 8_589_934_592 with six exact
    decimal places. This is the largest magnitude at which a float64 still
    resolves a 1e-6 step (its ULP stays below 1e-6); beyond it the sixth
    decimal digit is no longer representable, so it is intentionally excluded
    from the safe range rather than silently rounded.
*/
package decint

import "errors"

const (
	// precision of the float-to-integer conversion (max 6 decimal digits).
	precision float64 = 1e+06

	// scale is the integer twin of precision, used for exact integer formatting.
	scale = 1_000_000

	// MaxInt is the maximum scaled integer that preserves six exact decimals
	// (2^33 * 1e6). It is below 2^53, so it is still an exact float64.
	MaxInt = 8_589_934_592_000_000

	// MaxFloat is the maximum value that preserves six exact decimals in a
	// float64 (2^33). Above it the float64 ULP exceeds 1e-6 and the sixth
	// decimal digit can no longer be represented.
	MaxFloat = 8_589_934_592
)

var (
	// ErrInvalidNumber indicates the input string is not a valid finite number.
	ErrInvalidNumber = errors.New("invalid decimal number")

	// ErrOutOfRange indicates the value is outside the safe range.
	ErrOutOfRange = errors.New("value out of safe range")
)
