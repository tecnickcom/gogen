package random

import (
	"testing"
)

func BenchmarkUID64(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		_ = r.UID64()
	}
}

func BenchmarkUID128(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		_ = r.UID128()
	}
}
