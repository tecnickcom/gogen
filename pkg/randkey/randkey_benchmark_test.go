package randkey

import (
	"testing"
)

func BenchmarkNew(b *testing.B) {
	b.ResetTimer()

	for range b.N {
		_ = New()
	}
}
