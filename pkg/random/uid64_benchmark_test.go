package random

import (
	"testing"
)

func BenchmarkRnd_UID64(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		sinkUID64 = r.UID64()
	}
}

func BenchmarkRnd_UID64_Format(b *testing.B) {
	u := New(nil).UID64()

	var dst [16]byte

	for b.Loop() {
		u.Format(&dst)
	}

	sinkByte = dst[0]
}

// BenchmarkRnd_UID64_Byte_NoEscape measures Byte when the returned slice stays
// local: the backing array is stack-allocated and the call is free.
func BenchmarkRnd_UID64_Byte_NoEscape(b *testing.B) {
	u := New(nil).UID64()

	var acc byte

	for b.Loop() {
		p := u.Byte()
		acc ^= p[0] ^ p[15]
	}

	sinkByte = acc
}

// BenchmarkRnd_UID64_Byte_Escaping measures Byte when the returned slice outlives
// the call: the backing array is heap-allocated. Both numbers are documented.
func BenchmarkRnd_UID64_Byte_Escaping(b *testing.B) {
	u := New(nil).UID64()

	for b.Loop() {
		sinkBytes = u.Byte()
	}
}

// The Hex and String benchmarks measure formatting only: generation is hoisted out
// of the loop so the entropy read does not mask the cost of the conversion.
func BenchmarkRnd_UID64_Hex(b *testing.B) {
	u := New(nil).UID64()

	for b.Loop() {
		sinkString = u.Hex()
	}
}

func BenchmarkRnd_UID64_String(b *testing.B) {
	u := New(nil).UID64()

	for b.Loop() {
		sinkString = u.String()
	}
}
