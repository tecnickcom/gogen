package randkey

import (
	"testing"
)

func BenchmarkNew(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = New()
	}
}
