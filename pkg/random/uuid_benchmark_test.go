package random

import (
	"testing"
)

func BenchmarkRnd_UUIDv7(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		_ = r.UUIDv7()
	}
}

func BenchmarkUUID_Format(b *testing.B) {
	u := New(nil).UUIDv7()

	var dst [36]byte

	for b.Loop() {
		u.Format(&dst)
	}
}

func BenchmarkUUID_Byte(b *testing.B) {
	u := New(nil).UUIDv7()

	for b.Loop() {
		_ = u.Byte()
	}
}

func BenchmarkUUID_String(b *testing.B) {
	u := New(nil).UUIDv7()

	for b.Loop() {
		_ = u.String()
	}
}
