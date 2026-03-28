/*
Package decint provides utility functions to parse and represent decimal values
as fixed-point integers with a defined precision.

This package solves the common problem of safely handling small monetary and
fixed-precision decimal values without using floating-point arithmetic for
storage or comparison.

Decint is designed for values with at most six decimal places, and it stores
those values as integers to avoid rounding issues and preserve exactness.

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
