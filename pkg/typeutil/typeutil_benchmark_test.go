package typeutil_test

import (
	"testing"

	"github.com/tecnickcom/gogen/pkg/typeutil"
)

func BenchmarkIsNil(b *testing.B) {
	var p *int

	v := any(p)

	b.ReportAllocs()

	for b.Loop() {
		_ = typeutil.IsNil(v)
	}
}

func BenchmarkIsZeroInt(b *testing.B) {
	v := 0

	b.ReportAllocs()

	for b.Loop() {
		_ = typeutil.IsZero(v)
	}
}

func BenchmarkIsZeroStruct(b *testing.B) {
	type sample struct {
		A, B int
		C    string
	}

	var v sample

	b.ReportAllocs()

	for b.Loop() {
		_ = typeutil.IsZero(v)
	}
}

func BenchmarkZero(b *testing.B) {
	v := 42

	b.ReportAllocs()

	for b.Loop() {
		_ = typeutil.Zero(v)
	}
}

func BenchmarkValue(b *testing.B) {
	v := 42

	b.ReportAllocs()

	for b.Loop() {
		_ = typeutil.Value(&v)
	}
}

func BenchmarkBoolToInt(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = typeutil.BoolToInt(true)
	}
}

func BenchmarkBoolToNumInt(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = typeutil.BoolToNum[int](true)
	}
}

func BenchmarkBoolToNumFloat64(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = typeutil.BoolToNum[float64](true)
	}
}
