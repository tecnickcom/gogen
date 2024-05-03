package random

import (
	"testing"
)

func BenchmarkRnd_RandUint32(b *testing.B) {
	b.ResetTimer()

	r := New(nil)

	for i := 0; i < b.N; i++ {
		_ = r.RandUint32()
	}
}

func BenchmarkRnd_RandUint64(b *testing.B) {
	b.ResetTimer()

	r := New(nil)

	for i := 0; i < b.N; i++ {
		_ = r.RandUint64()
	}
}

func BenchmarkRnd_RandString(b *testing.B) {
	b.ResetTimer()

	r := New(nil)

	for i := 0; i < b.N; i++ {
		_, _ = r.RandString(16)
	}
}
