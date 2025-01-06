package stringmetric

import (
	"testing"
)

func BenchmarkDLDistance(b *testing.B) {
	b.ResetTimer()

	for range b.N {
		_ = DLDistance("intention", "execution")
	}
}
