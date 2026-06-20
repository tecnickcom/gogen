package sliceutil

import (
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"
)

func TestStats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    []int
		want    *DescStats[int]
		wantErr bool
	}{
		{
			name:    "Nil input",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Empty input",
			data:    []int{},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "One zero input",
			data:    []int{0},
			want:    &DescStats[int]{Count: 1, Entropy: 0, ExKurtosis: 0, Max: 0, MaxID: 0, Mean: 0, MeanDev: 0, Median: 0, Min: 0, MinID: 0, Mode: 0, ModeFreq: 1, Range: 0, Skewness: 0, StdDev: 0, Sum: 0, Variance: 0},
			wantErr: false,
		},
		{
			name:    "One element",
			data:    []int{13},
			want:    &DescStats[int]{Count: 1, Entropy: 0, ExKurtosis: 0, Max: 13, MaxID: 0, Mean: 13, MeanDev: 0, Median: 13, Min: 13, MinID: 0, Mode: 13, ModeFreq: 1, Range: 0, Skewness: 0, StdDev: 0, Sum: 13, Variance: 0},
			wantErr: false,
		},
		{
			name:    "Two elements",
			data:    []int{29, 13},
			want:    &DescStats[int]{Count: 2, Entropy: 0.6187, ExKurtosis: 0, Max: 29, MaxID: 0, Mean: 21, MeanDev: 0, Median: 21, Min: 13, MinID: 1, Mode: 29, ModeFreq: 1, Range: 16, Skewness: 0, StdDev: 11.3137, Sum: 42, Variance: 128},
			wantErr: false,
		},
		{
			name:    "Three elements",
			data:    []int{13, 37, 29},
			want:    &DescStats[int]{Count: 3, Entropy: 1.0200, ExKurtosis: 0, Max: 37, MaxID: 1, Mean: 26.3333, MeanDev: 1.1842e-15, Median: 29, Min: 13, MinID: 0, Mode: 13, ModeFreq: 1, Range: 24, Skewness: -0.9352, StdDev: 12.2202, Sum: 79, Variance: 149.3333},
			wantErr: false,
		},
		{
			name:    "Four elements",
			data:    []int{53, 13, 37, 29},
			want:    &DescStats[int]{Count: 4, Entropy: 1.2841, ExKurtosis: 0.3905, Max: 53, MaxID: 0, Mean: 33, MeanDev: 0, Median: 33, Min: 13, MinID: 1, Mode: 53, ModeFreq: 1, Range: 40, Skewness: 0, StdDev: 16.6533, Sum: 132, Variance: 277.3333},
			wantErr: false,
		},
		{
			name:    "Five elements",
			data:    []int{53, 13, 79, 37, 29},
			want:    &DescStats[int]{Count: 5, Entropy: 1.4645, ExKurtosis: 0.1751, Max: 79, MaxID: 2, Mean: 42.2, MeanDev: -2.8421e-15, Median: 37, Min: 13, MinID: 1, Mode: 53, ModeFreq: 1, Range: 66, Skewness: 0.6242, StdDev: 25.1236, Sum: 211, Variance: 631.2},
			wantErr: false,
		},
		{
			name:    "Six elements",
			data:    []int{53, 83, 13, 79, 37, 29},
			want:    &DescStats[int]{Count: 6, Entropy: 1.6462, ExKurtosis: -1.6680, Max: 83, MaxID: 1, Mean: 49, MeanDev: 0, Median: 45, Min: 13, MinID: 2, Mode: 53, ModeFreq: 1, Range: 70, Skewness: 0.1368, StdDev: 27.9714, Sum: 294, Variance: 782.4},
			wantErr: false,
		},
		{
			name:    "General case",
			data:    []int{53, 83, 13, 79, 13, 37, 83, 29, 37, 13, 83, 83},
			want:    &DescStats[int]{Count: 12, Entropy: 2.3019, ExKurtosis: -1.9100, Max: 83, MaxID: 1, Mean: 50.5, MeanDev: 0, Median: 45, Min: 13, MinID: 2, Mode: 83, ModeFreq: 4, Range: 70, Skewness: -0.0494, StdDev: 30.2850, Sum: 606, Variance: 917.1818},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Stats(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}

			require.True(t, cmp.Equal(tt.want, got, cmpopts.EquateApprox(0, 0.001)), got)
		})
	}
}

// requireFiniteStats asserts that every float64 field of a DescStats is finite.
func requireFiniteStats[V interface {
	~int | ~int64 | ~float64
}](t *testing.T, ds *DescStats[V]) {
	t.Helper()

	for name, f := range map[string]float64{
		"Entropy":    ds.Entropy,
		"ExKurtosis": ds.ExKurtosis,
		"Mean":       ds.Mean,
		"MeanDev":    ds.MeanDev,
		"Median":     ds.Median,
		"Skewness":   ds.Skewness,
		"StdDev":     ds.StdDev,
		"Variance":   ds.Variance,
	} {
		require.Falsef(t, math.IsNaN(f), "field %s is NaN", name)
		require.Falsef(t, math.IsInf(f, 0), "field %s is Inf", name)
	}
}

func TestStatsIntZeroSumAndNegative(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []int
	}{
		{
			name: "all zeros",
			data: []int{0, 0, 0, 0},
		},
		{
			name: "zero sum mixed sign",
			data: []int{-5, 5, -3, 3},
		},
		{
			name: "negative sum",
			data: []int{-1, -2, -3, -4, -5},
		},
		{
			name: "all negative",
			data: []int{-10, -20, -30},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Stats(tt.data)
			require.NoError(t, err)
			require.NotNil(t, got)
			// entropy is reported as 0 (not NaN/Inf) when the sum is not strictly positive
			require.Zero(t, got.Entropy)
			requireFiniteStats(t, got)
		})
	}
}

func TestStatsFloatDatasets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []float64
	}{
		{
			name: "positive floats with positive sum",
			data: []float64{1.5, 2.5, 3.5, 4.5},
		},
		{
			name: "floats with zero sum",
			data: []float64{-2.5, 2.5, -1.0, 1.0},
		},
		{
			name: "floats with negative sum",
			data: []float64{-1.5, -2.5, 0.5},
		},
		{
			name: "mixed sign positive sum",
			data: []float64{-1.0, 10.0, -2.0, 5.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Stats(tt.data)
			require.NoError(t, err)
			require.NotNil(t, got)
			requireFiniteStats(t, got)
		})
	}
}
