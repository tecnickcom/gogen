package stringkey

import (
	"testing"
)

func BenchmarkNew(b *testing.B) {
	for b.Loop() {
		_ = New("", "a", "abcdef1234", "学院路30号", " ăâîșț  ĂÂÎȘȚ  ") //nolint:gosmopolitan
	}
}
