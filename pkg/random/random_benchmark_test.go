package random

import (
	"testing"
)

func BenchmarkRnd_RandomBytes(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		_, _ = r.RandomBytes(16)
	}
}

func BenchmarkRnd_RandUint32(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		_ = r.RandUint32()
	}
}

func BenchmarkRnd_RandUint64(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		_ = r.RandUint64()
	}
}

func BenchmarkRnd_RandHex64(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		_ = r.RandHex64()
	}
}

func BenchmarkRnd_RandString64(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		_ = r.RandString64()
	}
}

func BenchmarkRnd_RandString(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		_, _ = r.RandString(16)
	}
}
