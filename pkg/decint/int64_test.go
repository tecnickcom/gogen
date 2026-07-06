package decint

import (
	"math"
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
		{
			name: "nan clamps to zero",
			v:    math.NaN(),
			want: 0,
		},
		{
			name: "positive infinity clamps to max",
			v:    math.Inf(1),
			want: MaxInt,
		},
		{
			name: "negative infinity clamps to min",
			v:    math.Inf(-1),
			want: -MaxInt,
		},
		{
			name: "over range clamps to max",
			v:    MaxFloat * 2,
			want: MaxInt,
		},
		{
			name: "under range clamps to min",
			v:    -MaxFloat * 2,
			want: -MaxInt,
		},
		{
			name: "rounds one-ULP-low float representation", // regression: truncation returned 8199999
			v:    8.2,
			want: 8200000,
		},
		{
			name: "rounds negative one-ULP-off float representation",
			v:    -8.2,
			want: -8200000,
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
		name  string
		v     string
		want  int64
		errIs error
	}{
		{
			name: "zero",
			v:    "0",
			want: 0,
		},
		{
			name: "max",
			v:    "8589934592",
			want: MaxInt,
		},
		{
			name: "min",
			v:    "-8589934592",
			want: -MaxInt,
		},
		{
			name:  "error",
			v:     "ERROR",
			want:  0,
			errIs: ErrInvalidNumber,
		},
		{
			name:  "nan returns invalid number error",
			v:     "NaN",
			want:  0,
			errIs: ErrInvalidNumber,
		},
		{
			name:  "positive infinity returns invalid number error",
			v:     "Inf",
			want:  0,
			errIs: ErrInvalidNumber,
		},
		{
			name:  "negative infinity returns invalid number error",
			v:     "-Inf",
			want:  0,
			errIs: ErrInvalidNumber,
		},
		{
			name:  "over range returns out of range error",
			v:     "9007199255",
			want:  0,
			errIs: ErrOutOfRange,
		},
		{
			name:  "under range returns out of range error",
			v:     "-9007199255",
			want:  0,
			errIs: ErrOutOfRange,
		},
		{
			name:  "just above max returns out of range error", // 2^33 + 1e-6, first float-unsafe value
			v:     "8589934592.000001",
			want:  0,
			errIs: ErrOutOfRange,
		},
		{
			name: "exact decimal rounds correctly", // regression: truncation returned 8199999
			v:    "8.2",
			want: 8200000,
		},
		{
			name: "rounds seventh decimal half away from zero up",
			v:    "1.2345675",
			want: 1234568,
		},
		{
			name: "rounds seventh decimal half away from zero down",
			v:    "1.2345665",
			want: 1234567,
		},
		{
			name: "rounds negative seventh decimal half away from zero",
			v:    "-1.2345675",
			want: -1234568,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := StringToInt(tt.v)

			if tt.errIs != nil {
				require.ErrorIs(t, err, tt.errIs)
			} else {
				require.NoError(t, err)
			}

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
			want: "8589934592.000000",
		},
		{
			name: "min",
			v:    -MaxInt,
			want: "-8589934592.000000",
		},
		{
			name: "just below max is exact", // regression: float64 formatting printed a wrong last digit
			v:    MaxInt - 1,
			want: "8589934591.999999",
		},
		{
			name: "just above negative min is exact",
			v:    -(MaxInt - 1),
			want: "-8589934591.999999",
		},
		{
			name: "exact decimal round-trip",
			v:    8200000,
			want: "8.200000",
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
