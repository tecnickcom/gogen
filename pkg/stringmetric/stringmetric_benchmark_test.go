package stringmetric

import (
	"testing"
)

func BenchmarkDLDistance(b *testing.B) {
	for b.Loop() {
		_ = DLDistance("intention", "execution")
	}
}
