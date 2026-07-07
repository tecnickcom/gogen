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

func BenchmarkRnd_UID64_Format(b *testing.B) {
	u := New(nil).UID64()

	var dst [16]byte

	for b.Loop() {
		u.Format(&dst)
	}
}

func BenchmarkRnd_UID64_Byte(b *testing.B) {
	u := New(nil).UID64()

	for b.Loop() {
		_ = u.Byte()
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
