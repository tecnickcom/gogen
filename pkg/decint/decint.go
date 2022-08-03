// Package decint provides utility functions to parse and represent decimal values as integer with a set precision.
// Typically this is useful to store small currency values.
// Safe values are up to 2^53 / 1e+6 = 9_007_199_254.740_992
package decint

const (
	// precision of the float-to-integer conversion (max 6 decimal digits).
	precision float64 = 1e+06

	stringFormat = "%.6f"

	// MaxInt is the maximum integer number that can be safely represented (2^53).
	MaxInt = 9_007_199_254_740_992

	// MaxFloat is the maximum float number that can be safely represented (2^53 / 1e+06).
	MaxFloat = 9_007_199_254.740_992
)
