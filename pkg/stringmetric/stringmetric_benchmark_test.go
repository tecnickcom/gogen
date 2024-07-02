package stringmetric

import (
	"testing"
)

func BenchmarkDLDistance(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = DLDistance("intention", "execution")
	}
}
