package filter

import "math"

// numericNaN is a sentinel returned by cmpFloat for NaN operands; callers treat it as "no ordering".
const numericNaN = 2

// numericKind classifies a normalized numeric value so that large int64/uint64
// values can be compared exactly instead of being lossily widened to float64.
type numericKind uint8

const (
	numericNone  numericKind = iota // not a numeric value
	numericInt                      // signed integer, stored in i
	numericUint                     // unsigned integer, stored in u
	numericFloat                    // floating point, stored in f
)

// numeric is a normalized numeric value preserving the exactness of integers.
// Integers are kept as int64/uint64 (not widened to float64) so that values
// beyond 2^53 still compare correctly for equality and ordering.
type numeric struct {
	kind numericKind
	i    int64
	u    uint64
	f    float64
}

// toNumeric normalizes any supported numeric value, reporting ok=false for non-numeric inputs.
//
//nolint:gocyclo,cyclop
func toNumeric(v any) (numeric, bool) {
	switch x := v.(type) {
	case int:
		return numeric{kind: numericInt, i: int64(x)}, true
	case int8:
		return numeric{kind: numericInt, i: int64(x)}, true
	case int16:
		return numeric{kind: numericInt, i: int64(x)}, true
	case int32:
		return numeric{kind: numericInt, i: int64(x)}, true
	case int64:
		return numeric{kind: numericInt, i: x}, true
	case uint:
		return numeric{kind: numericUint, u: uint64(x)}, true
	case uint8:
		return numeric{kind: numericUint, u: uint64(x)}, true
	case uint16:
		return numeric{kind: numericUint, u: uint64(x)}, true
	case uint32:
		return numeric{kind: numericUint, u: uint64(x)}, true
	case uint64:
		return numeric{kind: numericUint, u: x}, true
	case float32:
		return numeric{kind: numericFloat, f: float64(x)}, true
	case float64:
		return numeric{kind: numericFloat, f: x}, true
	}

	return numeric{}, false
}

// float returns the value as a float64. It is only ever called on integer- or float-kinded values.
func (n numeric) float() float64 {
	if n.kind == numericFloat {
		return n.f
	}

	if n.kind == numericUint {
		return float64(n.u)
	}

	return float64(n.i)
}

// equals reports whether two normalized numeric values are exactly equal.
func (n numeric) equals(o numeric) bool {
	c, ok := n.compare(o)

	return ok && c == 0
}

// compare returns -1, 0 or 1 when n is less than, equal to or greater than o.
// ok is false when either operand is non-numeric or a NaN (so no ordering applies).
func (n numeric) compare(o numeric) (int, bool) {
	if n.kind == numericNone || o.kind == numericNone {
		return 0, false
	}

	// Exact integer comparison when neither side is a float.
	if n.kind != numericFloat && o.kind != numericFloat {
		return n.compareInt(o), true
	}

	c := cmpFloat(n.float(), o.float())
	if c == numericNaN {
		return 0, false
	}

	return c, true
}

// compareInt compares two integer-kinded numerics exactly, handling signed/unsigned mixes.
func (n numeric) compareInt(o numeric) int {
	switch {
	case n.kind == numericInt && o.kind == numericInt:
		return cmpInt64(n.i, o.i)
	case n.kind == numericUint && o.kind == numericUint:
		return cmpUint64(n.u, o.u)
	case n.kind == numericInt:
		return cmpIntUint(n.i, o.u)
	default:
		return -cmpIntUint(o.i, n.u)
	}
}

// cmpIntUint compares a signed and an unsigned integer exactly.
func cmpIntUint(i int64, u uint64) int {
	if i < 0 {
		return -1
	}

	return cmpUint64(uint64(i), u)
}

// cmpInt64 returns the sign of a-b for signed integers.
func cmpInt64(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// cmpUint64 returns the sign of a-b for unsigned integers.
func cmpUint64(a, b uint64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// cmpFloat returns the sign of a-b for float64 values, or numericNaN when either operand is NaN.
func cmpFloat(a, b float64) int {
	switch {
	case math.IsNaN(a) || math.IsNaN(b):
		return numericNaN
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
