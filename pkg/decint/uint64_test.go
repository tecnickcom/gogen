package decint

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFloatToUint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		v    float64
		want uint64
	}{
		{
			name: "zero",
			v:    0,
			want: 0,
		},
		{
			name: "max",
			v:    MaxFloat,
			want: MaxInt,
		},
		{
			name: "min",
			v:    -MaxFloat,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := FloatToUint(tt.v)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestUintToFloat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		v    uint64
		want float64
	}{
		{
			name: "zero",
			v:    0,
			want: 0,
		},
		{
			name: "max",
			v:    MaxInt, // 2^53
			want: MaxFloat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := UintToFloat(tt.v)
			require.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestStringToUint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		v       string
		want    uint64
		wantErr bool
	}{
		{
			name: "zero",
			v:    "0",
			want: 0,
		},
		{
			name: "max",
			v:    "9007199254.740992",
			want: MaxInt,
		},
		{
			name: "min",
			v:    "-9007199254.740992",
			want: 0,
		},
		{
			name:    "error",
			v:       "ERROR",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := StringToUint(tt.v)

			require.Equal(t, tt.wantErr, err != nil)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestUintToString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		v    uint64
		want string
	}{
		{
			name: "zero",
			v:    0,
			want: "0.000000",
		},
		{
			name: "max",
			v:    MaxInt,
			want: "9007199254.740992",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := UintToString(tt.v)
			require.Equal(t, tt.want, got)
		})
	}
}
