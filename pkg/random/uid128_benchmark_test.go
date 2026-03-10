package random

import (
	"testing"
)

func BenchmarkRnd_UID128(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		_ = r.UID128()
	}
}

func BenchmarkRnd_UID128_Hex(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		_ = r.UID128().Hex()
	}
}

func BenchmarkRnd_UID128_String(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		_ = r.UID128().String()
	}
}
