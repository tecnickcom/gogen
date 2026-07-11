package decint_test

import (
	"testing"

	"github.com/tecnickcom/nurago/pkg/decint"
)

func BenchmarkFloatToInt(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = decint.FloatToInt(123456.789012)
	}
}

func BenchmarkIntToFloat(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = decint.IntToFloat(123456789012)
	}
}

func BenchmarkStringToInt(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_, _ = decint.StringToInt("123456.789012")
	}
}

func BenchmarkIntToString(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = decint.IntToString(123456789012)
	}
}

func BenchmarkFloatToUint(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = decint.FloatToUint(123456.789012)
	}
}

func BenchmarkUintToFloat(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = decint.UintToFloat(123456789012)
	}
}

func BenchmarkStringToUint(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_, _ = decint.StringToUint("123456.789012")
	}
}

func BenchmarkUintToString(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = decint.UintToString(123456789012)
	}
}
