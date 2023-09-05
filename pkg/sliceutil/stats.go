package sliceutil

import (
	"fmt"
	"math"
	"slices"

	"github.com/Vonage/gosrvlib/pkg/typeutil"
)

// DescStats contains descriptive statistics items for a data set.
type DescStats[V typeutil.Number] struct {
	// Count is the total number of items in the data set.
	Count int `json:"count"`

	// Entropy computes the Shannon entropy of a distribution.
	Entropy float64 `json:"entropy"`

	// ExKurtosis is the population excess kurtosis of the data set.
	// The kurtosis is defined by the 4th moment of the mean divided by the squared variance.
	// The excess kurtosis subtracts 3.0 so that the excess kurtosis of the normal distribution is zero.
	ExKurtosis float64 `json:"exkurtosis"`

	// Max is the maximum value of the data.
	Max V `json:"max"`

	// MaxID is the index (key) of the Max malue in a data set.
	MaxID int `json:"maxid"`

	// Mean or Average is a central tendency of the data.
	Mean float64 `json:"mean"`

	// MeanDev is the Mean Deviation or Mean Absolute Deviation.
	// It is an average of absolute differences between each value in the data, and the average of all values.
	MeanDev float64 `json:"meandev"`

	// Median is the value that divides the data into 2 equal parts.
	// When the data is sorted, the number of terms on the left and right side of median is the same.
	Median float64 `json:"median"`

	// Min is the minimal value of the data.
	Min V `json:"min"`

	// MinID is the index (key) of the Min malue in a data set.
	MinID int `json:"minid"`

	// Mode is the term appearing maximum time in data set.
	// It is the term that has the highest frequency.
	Mode V `json:"mode"`

	// ModeFreq is the frequency of the Mode value.
	ModeFreq int `json:"modefreq"`

	// Range is the difference between the highest (Max) and lowest (Min) value.
	Range V `json:"range"`

	// Skewness is a measure of the asymmetry of the probability distribution of a real-valued random variable about its mean.
	// Provides the adjusted Fisher-Pearson standardized moment coefficient.
	Skewness float64 `json:"skewness"`

	// StdDev is the Standard deviation of the data.
	// It measures the average distance between each quantity and mean.
	StdDev float64 `json:"stddev"`

	// Sum of all the values in the data.
	Sum V `json:"sum"`

	// Variance is a square of average distance between each quantity and Mean.
	Variance float64 `json:"variance"`
}

// Stats returns descriptive statistics parameters to summarize the input data set.
//
//nolint:gocognit,gocyclo
func Stats[S ~[]V, V typeutil.Number](s S) (*DescStats[V], error) {
	n := len(s)

	if n < 1 {
		return nil, fmt.Errorf("input slice is empty")
	}

	ord := slices.Clone(s)
	slices.Sort(ord)

	ds := &DescStats[V]{
		Count:    len(s),
		Max:      s[0],
		Median:   float64(s[0]),
		Min:      s[0],
		Mode:     s[0],
		ModeFreq: 1,
		Sum:      s[0],
		Mean:     float64(s[0]),
	}

	if n == 1 {
		return ds, nil
	}

	nf := float64(n)
	freq := 1

	for i := 1; i < n; i++ {
		v := s[i]
		vf := float64(s[i])

		ds.Sum += v

		if v < ds.Min {
			ds.Min = v
			ds.MinID = i
		}

		if v > ds.Max {
			ds.Max = v
			ds.MaxID = i
		}

		if v != 0 {
			ds.Entropy -= vf * math.Log(vf)
		}

		if ord[i] == ord[i-1] {
			freq++
		} else {
			if freq > ds.ModeFreq {
				ds.Mode = ord[i]
				ds.ModeFreq = freq
			}
			freq = 1
		}
	}

	if freq > ds.ModeFreq {
		ds.Mode = ord[n-1]
		ds.ModeFreq = freq
	}

	ds.Range = ds.Max - ds.Min
	ds.Mean = float64(ds.Sum) / nf

	midpos := n / 2
	if n%2 != 0 {
		ds.Median = float64(ord[midpos])
	} else {
		ds.Median = (float64(ord[midpos-1]) + float64(ord[midpos])) / 2
	}

	for i := 0; i < n; i++ {
		d := float64(ord[i]) - ds.Mean
		ds.MeanDev += d
		ds.Variance += d * d
	}

	ds.MeanDev /= nf
	ds.Variance /= (nf - 1)
	ds.StdDev = math.Sqrt(ds.Variance)

	if n < 3 {
		return ds, nil
	}

	for i := 0; i < n; i++ {
		d := (float64(ord[i]) - ds.Mean) / ds.StdDev
		d3 := d * d * d
		ds.Skewness += d3
		ds.ExKurtosis += d3 * d
	}

	ds.Skewness *= (nf / ((nf - 1) * (nf - 2))) // adjusted Fisher-Pearson standardized moment coefficient

	if n < 4 {
		ds.ExKurtosis = 0
	} else {
		ds.ExKurtosis = (ds.ExKurtosis * (((nf + 1) / (nf - 1)) * (nf / (nf - 2)) * (1 / (nf - 3)))) - (3 * ((nf - 1) / (nf - 2)) * ((nf - 1) / (nf - 3)))
	}

	return ds, nil
}
