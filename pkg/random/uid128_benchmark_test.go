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

func BenchmarkRnd_UID128_Format(b *testing.B) {
	u := New(nil).UID128()

	var dst [32]byte

	for b.Loop() {
		u.Format(&dst)
	}
}

func BenchmarkRnd_UID128_Byte(b *testing.B) {
	u := New(nil).UID128()

	for b.Loop() {
		_ = u.Byte()
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
