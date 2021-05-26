package decint

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	maxInt   = 9_007_199_254_740_992 // = 2^53
	maxFloat = 9_007_199_254.740_992 // = 2^53 / 1e+06
)

func TestFloatToInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		v    float64
		want int64
	}{
		{
			name: "zero",
			v:    0,
			want: 0,
		},
		{
			name: "max",
			v:    maxFloat,
			want: maxInt,
		},
		{
			name: "min",
			v:    -maxFloat,
			want: -maxInt,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := FloatToInt(tt.v)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestIntToFloat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		v    int64
		want float64
	}{
		{
			name: "zero",
			v:    0,
			want: 0,
		},
		{
			name: "max",
			v:    maxInt,
			want: maxFloat,
		},
		{
			name: "min",
			v:    -maxInt,
			want: -maxFloat,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := IntToFloat(tt.v)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestStringToInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		v       string
		want    int64
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
			want: maxInt,
		},
		{
			name: "min",
			v:    "-9007199254.740992",
			want: -maxInt,
		},
		{
			name:    "error",
			v:       "ERROR",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := StringToInt(tt.v)

			require.Equal(t, tt.wantErr, err != nil)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestIntToString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		v    int64
		want string
	}{
		{
			name: "zero",
			v:    0,
			want: "0.000000",
		},
		{
			name: "max",
			v:    maxInt,
			want: "9007199254.740992",
		},
		{
			name: "min",
			v:    -maxInt,
			want: "-9007199254.740992",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := IntToString(tt.v)
			require.Equal(t, tt.want, got)
		})
	}
}
