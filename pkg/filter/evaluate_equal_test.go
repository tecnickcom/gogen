package filter

import (
	"math"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEqual_Evaluate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		ref   any
		value any
		want  bool
	}{
		{
			name:  "true - int / int",
			ref:   42,
			value: 42,
			want:  true,
		},
		{
			name:  "true - float64 / int",
			ref:   42.0,
			value: 42,
			want:  true,
		},
		{
			name:  "true - int / float64",
			ref:   42,
			value: 42.0,
			want:  true,
		},
		{
			name:  "true - float64 / float64",
			ref:   42.0,
			value: 42.0,
			want:  true,
		},
		{
			name:  "true - int8 / float64",
			ref:   int8(42),
			value: 42.0,
			want:  true,
		},
		{
			name:  "true - int16 / float64",
			ref:   int16(42),
			value: 42.0,
			want:  true,
		},
		{
			name:  "true - int32 / float64",
			ref:   int32(42),
			value: 42.0,
			want:  true,
		},
		{
			name:  "true - int64 / float64",
			ref:   int64(42),
			value: 42.0,
			want:  true,
		},
		{
			name:  "true - uint / float64",
			ref:   uint(42),
			value: 42.0,
			want:  true,
		},
		{
			name:  "true - uint8 / float64",
			ref:   uint8(42),
			value: 42.0,
			want:  true,
		},
		{
			name:  "true - uint16 / float64",
			ref:   uint16(42),
			value: 42.0,
			want:  true,
		},
		{
			name:  "true - uint32 / float64",
			ref:   uint32(42),
			value: 42.0,
			want:  true,
		},
		{
			name:  "true - uint64 / float64",
			ref:   uint64(42),
			value: 42.0,
			want:  true,
		},
		{
			name:  "true - float32 / float64",
			ref:   float32(42),
			value: 42.0,
			want:  true,
		},
		{
			name:  "false - int / int",
			ref:   42,
			value: 43,
			want:  false,
		},
		{
			name:  "false - float64 / int",
			ref:   42.1,
			value: 42,
			want:  false,
		},
		{
			name:  "false - float64 / float64",
			ref:   42.0,
			value: 42.1,
			want:  false,
		},
		{
			name:  "false - uint8 / string",
			ref:   uint8(42),
			value: "42",
			want:  false,
		},
		{
			name:  "false - string / string",
			ref:   "ciao",
			value: "hello",
			want:  false,
		},
		{
			name:  "true - string / string",
			ref:   "hello",
			value: "hello",
			want:  true,
		},
		{
			name:  "false - case string / string",
			ref:   "HELLO",
			value: "hello",
			want:  false,
		},
		{
			name:  "true - nil / nil",
			ref:   nil,
			value: nil,
			want:  true,
		},
		{
			name:  "true - string / stringAlias",
			ref:   "hello",
			value: stringAlias("hello"),
			want:  true,
		},
		{
			name:  "false - string / stringAlias",
			ref:   "hello",
			value: stringAlias("world"),
			want:  false,
		},
		{
			// Beyond 2^53 float64 cannot distinguish adjacent integers; exact compare must.
			name:  "true - large int64 exact",
			ref:   int64(1)<<53 + 1,
			value: int64(1)<<53 + 1,
			want:  true,
		},
		{
			name:  "false - large int64 off by one",
			ref:   int64(1)<<53 + 1,
			value: int64(1)<<53 + 2,
			want:  false,
		},
		{
			name:  "true - large uint64 exact",
			ref:   uint64(math.MaxUint64),
			value: uint64(math.MaxUint64),
			want:  true,
		},
		{
			name:  "false - large uint64 off by one",
			ref:   uint64(math.MaxUint64),
			value: uint64(math.MaxUint64) - 1,
			want:  false,
		},
		{
			name:  "false - numeric ref vs nil value",
			ref:   int64(1) << 60,
			value: nil,
			want:  false,
		},
		{
			// Non-comparable dynamic types (e.g. JSON objects) must not panic.
			name:  "true - uncomparable map / map (deep equal)",
			ref:   map[string]any{"a": 1.0},
			value: map[string]any{"a": 1.0},
			want:  true,
		},
		{
			name:  "false - uncomparable map / map",
			ref:   map[string]any{"a": 1.0},
			value: map[string]any{"a": 2.0},
			want:  false,
		},
		{
			name:  "true - uncomparable slice / slice (deep equal)",
			ref:   []any{"x"},
			value: []any{"x"},
			want:  true,
		},
		{
			name:  "false - uncomparable slice / string",
			ref:   []any{"x"},
			value: "x",
			want:  false,
		},
		{
			// Comparable non-string/non-numeric references (e.g. bool) fall back to the
			// interface "==" comparison.
			name:  "true - bool / bool",
			ref:   true,
			value: true,
			want:  true,
		},
		{
			name:  "false - bool / bool",
			ref:   true,
			value: false,
			want:  false,
		},
		{
			name:  "false - map ref vs nil value",
			ref:   map[string]any{"a": 1.0},
			value: nil,
			want:  false,
		},
		{
			// A typed-nil value must not equal a non-nil reference of a different type.
			name:  "false - map ref vs typed-nil slice value",
			ref:   map[string]any{"a": 1.0},
			value: []any(nil),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res := newEqual(tt.ref).Evaluate(reflect.ValueOf(tt.value))
			require.Equal(t, tt.want, res)
		})
	}
}
