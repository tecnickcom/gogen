package filter

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToNumeric(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    any
		wantOK   bool
		wantKind numericKind
	}{
		{name: "int", value: int(1), wantOK: true, wantKind: numericInt},
		{name: "int8", value: int8(1), wantOK: true, wantKind: numericInt},
		{name: "int16", value: int16(1), wantOK: true, wantKind: numericInt},
		{name: "int32", value: int32(1), wantOK: true, wantKind: numericInt},
		{name: "int64", value: int64(1), wantOK: true, wantKind: numericInt},
		{name: "uint", value: uint(1), wantOK: true, wantKind: numericUint},
		{name: "uint8", value: uint8(1), wantOK: true, wantKind: numericUint},
		{name: "uint16", value: uint16(1), wantOK: true, wantKind: numericUint},
		{name: "uint32", value: uint32(1), wantOK: true, wantKind: numericUint},
		{name: "uint64", value: uint64(1), wantOK: true, wantKind: numericUint},
		{name: "float32", value: float32(1), wantOK: true, wantKind: numericFloat},
		{name: "float64", value: float64(1), wantOK: true, wantKind: numericFloat},
		{name: "string", value: "1", wantOK: false, wantKind: numericNone},
		{name: "nil", value: nil, wantOK: false, wantKind: numericNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			n, ok := toNumeric(tt.value)
			require.Equal(t, tt.wantOK, ok)
			require.Equal(t, tt.wantKind, n.kind)
		})
	}
}

func TestNumericCompare(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        any
		b        any
		wantSign int
		wantOK   bool
	}{
		// exact integer comparisons, including values beyond 2^53.
		{name: "int64 equal large", a: int64(1) << 60, b: int64(1) << 60, wantSign: 0, wantOK: true},
		{name: "int64 lt large", a: int64(1)<<53 + 1, b: int64(1)<<53 + 2, wantSign: -1, wantOK: true},
		{name: "int64 gt large", a: int64(1)<<53 + 2, b: int64(1)<<53 + 1, wantSign: 1, wantOK: true},
		{name: "uint64 equal large", a: uint64(1) << 63, b: uint64(1) << 63, wantSign: 0, wantOK: true},
		{name: "uint64 lt", a: uint64(1), b: uint64(2), wantSign: -1, wantOK: true},
		{name: "uint64 gt", a: uint64(3), b: uint64(2), wantSign: 1, wantOK: true},
		// signed/unsigned mixes.
		{name: "int neg vs uint", a: int64(-1), b: uint64(1), wantSign: -1, wantOK: true},
		{name: "uint vs int neg", a: uint64(1), b: int64(-1), wantSign: 1, wantOK: true},
		{name: "int pos vs uint equal", a: int64(7), b: uint64(7), wantSign: 0, wantOK: true},
		{name: "int pos vs uint gt", a: int64(8), b: uint64(7), wantSign: 1, wantOK: true},
		{name: "uint vs int pos lt", a: uint64(6), b: int64(7), wantSign: -1, wantOK: true},
		// float involvement.
		{name: "float vs int lt", a: 1.5, b: 2, wantSign: -1, wantOK: true},
		{name: "int vs float gt", a: 3, b: 2.5, wantSign: 1, wantOK: true},
		{name: "float vs uint equal", a: 7.0, b: uint64(7), wantSign: 0, wantOK: true},
		// NaN yields no ordering.
		{name: "nan lhs", a: math.NaN(), b: 1, wantSign: 0, wantOK: false},
		{name: "nan rhs", a: 1, b: math.NaN(), wantSign: 0, wantOK: false},
		// non-numeric yields no ordering.
		{name: "none lhs", a: "x", b: 1, wantSign: 0, wantOK: false},
		{name: "none rhs", a: 1, b: "x", wantSign: 0, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			na, _ := toNumeric(tt.a)
			nb, _ := toNumeric(tt.b)

			sign, ok := na.compare(nb)
			require.Equal(t, tt.wantOK, ok)

			if tt.wantOK {
				require.Equal(t, tt.wantSign, sign)
				require.Equal(t, tt.wantSign == 0, na.equals(nb))
			}
		})
	}
}
