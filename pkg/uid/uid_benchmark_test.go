package uid

import (
	"testing"
)

func BenchmarkNewID64(b *testing.B) {
	b.ResetTimer()

	for range b.N {
		_ = NewID64()
	}
}

func BenchmarkNewID128(b *testing.B) {
	b.ResetTimer()

	for range b.N {
		_ = NewID128()
	}
}
