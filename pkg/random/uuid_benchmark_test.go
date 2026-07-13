package random

import (
	"testing"
)

func BenchmarkRnd_UUIDv7(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		sinkUUID = r.UUIDv7()
	}
}

func BenchmarkUUID_Format(b *testing.B) {
	u := New(nil).UUIDv7()

	var dst [36]byte

	for b.Loop() {
		u.Format(&dst)
	}

	sinkByte = dst[0]
}

// BenchmarkUUID_Byte_NoEscape measures Byte when the returned slice stays local:
// the backing array is stack-allocated and the call is free.
func BenchmarkUUID_Byte_NoEscape(b *testing.B) {
	u := New(nil).UUIDv7()

	var acc byte

	for b.Loop() {
		p := u.Byte()
		acc ^= p[0] ^ p[35]
	}

	sinkByte = acc
}

// BenchmarkUUID_Byte_Escaping measures Byte when the returned slice outlives the
// call: the backing array is heap-allocated. Both numbers are documented.
func BenchmarkUUID_Byte_Escaping(b *testing.B) {
	u := New(nil).UUIDv7()

	for b.Loop() {
		sinkBytes = u.Byte()
	}
}

func BenchmarkUUID_String(b *testing.B) {
	u := New(nil).UUIDv7()

	for b.Loop() {
		sinkString = u.String()
	}
}
