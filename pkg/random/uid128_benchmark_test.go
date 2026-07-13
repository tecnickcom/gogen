package random

import (
	"testing"
)

func BenchmarkRnd_UID128(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		sinkUID128 = r.UID128()
	}
}

func BenchmarkRnd_UID128_Format(b *testing.B) {
	u := New(nil).UID128()

	var dst [32]byte

	for b.Loop() {
		u.Format(&dst)
	}

	sinkByte = dst[0]
}

// BenchmarkRnd_UID128_Byte_NoEscape measures Byte when the returned slice stays
// local: the backing array is stack-allocated and the call is free.
func BenchmarkRnd_UID128_Byte_NoEscape(b *testing.B) {
	u := New(nil).UID128()

	var acc byte

	for b.Loop() {
		p := u.Byte()
		acc ^= p[0] ^ p[31]
	}

	sinkByte = acc
}

// BenchmarkRnd_UID128_Byte_Escaping measures Byte when the returned slice outlives
// the call: the backing array is heap-allocated. Both numbers are documented.
func BenchmarkRnd_UID128_Byte_Escaping(b *testing.B) {
	u := New(nil).UID128()

	for b.Loop() {
		sinkBytes = u.Byte()
	}
}

// The Hex and String benchmarks measure formatting only: generation is hoisted out
// of the loop so the entropy read does not mask the cost of the conversion.
func BenchmarkRnd_UID128_Hex(b *testing.B) {
	u := New(nil).UID128()

	for b.Loop() {
		sinkString = u.Hex()
	}
}

func BenchmarkRnd_UID128_String(b *testing.B) {
	u := New(nil).UID128()

	for b.Loop() {
		sinkString = u.String()
	}
}
