package random

import (
	"testing"
)

func BenchmarkRnd_UID64(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		_ = r.UID64()
	}
}

func BenchmarkRnd_UID64_Hex(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		_ = r.UID64().Hex()
	}
}

func BenchmarkRnd_UID64_String(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		_ = r.UID64().String()
	}
}
