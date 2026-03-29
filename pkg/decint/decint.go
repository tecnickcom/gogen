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

  - float-to-integer conversion uses scale-and-cast semantics, which truncates
    extra fractional digits toward zero instead of rounding.

Key benefits:

- deterministic decimal handling for currency, rates, and small amounts
- safe integer representation up to 2^53 with 6 decimal digits
- simplified serialization and comparison of fixed-precision values

Safe decimal values are limited up to 2^53 / 1e+6 = 9_007_199_254.740_992.
*/
package decint

const (
	// precision of the float-to-integer conversion (max 6 decimal digits).
	precision float64 = 1e+06

	// stringFormat is the verb used to print a 6-decimal digit float.
	stringFormat = "%.6f"

	// MaxInt is the maximum integer number that can be safely represented (2^53).
	MaxInt = 9_007_199_254_740_992

	// MaxFloat is the maximum float number that can be safely represented (2^53 / 1e+06).
	MaxFloat = 9_007_199_254.740_992
)
