package stringkey

import (
	"testing"
)

func BenchmarkNew(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = New("", "a", "abcdef1234", "学院路30号", " ăâîșț  ĂÂÎȘȚ  ")
	}
}
