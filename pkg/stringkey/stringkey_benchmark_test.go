package stringkey

import (
	"testing"
)

func BenchmarkNew(b *testing.B) {
	b.ResetTimer()

	for range b.N {
		_ = New("", "a", "abcdef1234", "学院路30号", " ăâîșț  ĂÂÎȘȚ  ")
	}
}
