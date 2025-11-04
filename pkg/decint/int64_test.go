package decint

import (
	"testing"

	"github.com/stretchr/testify/require"
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
			v:    MaxFloat,
			want: MaxInt,
		},
		{
			name: "min",
			v:    -MaxFloat,
			want: -MaxInt,
		},
	}

	for _, tt := range tests {
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
			v:    MaxInt,
			want: MaxFloat,
		},
		{
			name: "min",
			v:    -MaxInt,
			want: -MaxFloat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := IntToFloat(tt.v)
			require.InDelta(t, tt.want, got, 0.001)
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
			want: MaxInt,
		},
		{
			name: "min",
			v:    "-9007199254.740992",
			want: -MaxInt,
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
			v:    MaxInt,
			want: "9007199254.740992",
		},
		{
			name: "min",
			v:    -MaxInt,
			want: "-9007199254.740992",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := IntToString(tt.v)
			require.Equal(t, tt.want, got)
		})
	}
}
