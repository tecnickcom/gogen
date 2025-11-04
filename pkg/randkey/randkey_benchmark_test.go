package randkey

import (
	"testing"
)

func BenchmarkNew(b *testing.B) {
	for b.Loop() {
		_ = New()
	}
}
