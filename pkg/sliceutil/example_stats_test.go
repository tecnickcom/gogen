package sliceutil_test

import (
	"fmt"
	"log"

	"github.com/Vonage/gosrvlib/pkg/sliceutil"
)

func ExampleStats() {
	data := []int{53, 83, 13, 79, 13, 37, 83, 29, 37, 13, 83, 83}

	ds, err := sliceutil.Stats(data)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Count:      %d\n", ds.Count)
	fmt.Printf("Entropy:    %.3f\n", ds.Entropy)
	fmt.Printf("ExKurtosis: %.3f\n", ds.ExKurtosis)
	fmt.Printf("Max:        %d\n", ds.Max)
	fmt.Printf("MaxID:      %d\n", ds.MaxID)
	fmt.Printf("Mean:       %.3f\n", ds.Mean)
	fmt.Printf("MeanDev:    %.3f\n", ds.MeanDev)
	fmt.Printf("Median:     %.3f\n", ds.Median)
	fmt.Printf("Min:        %d\n", ds.Min)
	fmt.Printf("MinID:      %d\n", ds.MinID)
	fmt.Printf("Mode:       %d\n", ds.Mode)
	fmt.Printf("ModeFreq:   %d\n", ds.ModeFreq)
	fmt.Printf("Range:      %d\n", ds.Range)
	fmt.Printf("Skewness:   %.3f\n", ds.Skewness)
	fmt.Printf("StdDev:     %.3f\n", ds.StdDev)
	fmt.Printf("Sum:        %d\n", ds.Sum)
	fmt.Printf("Variance:   %.3f\n", ds.Variance)

	// Output:
	// Count:      12
	// Entropy:    -2277.134
	// ExKurtosis: -1.910
	// Max:        83
	// MaxID:      1
	// Mean:       50.500
	// MeanDev:    0.000
	// Median:     45.000
	// Min:        13
	// MinID:      2
	// Mode:       83
	// ModeFreq:   4
	// Range:      70
	// Skewness:   -0.049
	// StdDev:     30.285
	// Sum:        606
	// Variance:   917.182
}
