package paging

import "testing"

// Package-level sinks keep the results observable so the compiler cannot
// dead-code-eliminate the work being measured.

//nolint:gochecknoglobals // benchmark sink that defeats dead-code elimination
var sinkPaging Paging

func BenchmarkNew(b *testing.B) {
	for b.Loop() {
		sinkPaging = New(3, 5, 17)
	}
}

//nolint:gochecknoglobals // benchmark sinks that defeat dead-code elimination
var (
	sinkOffset uint
	sinkLimit  uint
)

func BenchmarkComputeOffsetAndLimit(b *testing.B) {
	for b.Loop() {
		sinkOffset, sinkLimit = ComputeOffsetAndLimit(3, 5)
	}
}
