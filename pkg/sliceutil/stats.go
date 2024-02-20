package sliceutil

import (
	"errors"
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
func Stats[S ~[]V, V typeutil.Number](s S) (*DescStats[V], error) {
	n := len(s)
	if n < 1 {
		return nil, errors.New("input slice is empty")
	}

	ds := &DescStats[V]{
		Count:    n,
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

	ord := slices.Clone(s)
	slices.Sort(ord)

	statsCenter(ds, s, ord, n, nf)
	statsVariability(ds, ord, nf)
	statsShape(ds, ord, nf)

	return ds, nil
}

// statsCenter calculates Min, Max, Mode,  ModeFreq, Range, Mean and Median.
func statsCenter[S ~[]V, V typeutil.Number](ds *DescStats[V], s, ord S, n int, nf float64) {
	freq := 1

	for i := 1; i < n; i++ {
		v := s[i]

		ds.Sum += v

		if v < ds.Min {
			ds.Min = v
			ds.MinID = i
		} else if v > ds.Max {
			ds.Max = v
			ds.MaxID = i
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

	statsMedian(ds, ord, n)
}

// statsMedian calculates Median.
func statsMedian[S ~[]V, V typeutil.Number](ds *DescStats[V], ord S, n int) {
	midpos := n / 2
	ds.Median = float64(ord[midpos])

	if n%2 == 0 {
		ds.Median = (float64(ord[midpos-1]) + ds.Median) / 2
	}
}

// statsVariability calculates Entropy, MeanDev, Varianceand  StdDev. It must be called after statsCenter().
func statsVariability[S ~[]V, V typeutil.Number](ds *DescStats[V], ord S, nf float64) {
	sum := float64(ds.Sum)

	for _, v := range ord {
		vf := float64(v)
		d := vf - ds.Mean
		ds.MeanDev += d
		ds.Variance += d * d

		if v != 0 {
			vf /= sum
			ds.Entropy -= vf * math.Log(vf)
		}
	}

	ds.MeanDev /= nf
	ds.Variance /= (nf - 1)
	ds.StdDev = math.Sqrt(ds.Variance)
}

// statsShape calculates Skewness and ExKurtosis. It must be called after statsVariability().
func statsShape[S ~[]V, V typeutil.Number](ds *DescStats[V], ord S, nf float64) {
	if nf < 3 {
		return
	}

	for _, v := range ord {
		d := (float64(v) - ds.Mean) / ds.StdDev
		d3 := d * d * d
		ds.Skewness += d3
		ds.ExKurtosis += d3 * d
	}

	ds.Skewness *= (nf / ((nf - 1) * (nf - 2))) // adjusted Fisher-Pearson standardized moment coefficient

	if nf < 4 {
		ds.ExKurtosis = 0
		return
	}

	ds.ExKurtosis = (ds.ExKurtosis * (((nf + 1) / (nf - 1)) * (nf / (nf - 2)) * (1 / (nf - 3)))) - (3 * ((nf - 1) / (nf - 2)) * ((nf - 1) / (nf - 3)))
}
