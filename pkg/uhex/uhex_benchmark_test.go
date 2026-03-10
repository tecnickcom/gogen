package uhex

import (
	"testing"
)

func BenchmarkHex64(b *testing.B) {
	u := uint64(0xffffffffffffffff)

	for b.Loop() {
		_ = Hex64(u)
	}
}

func BenchmarkHex32(b *testing.B) {
	u := uint32(0xffffffff)

	for b.Loop() {
		_ = Hex32(u)
	}
}

func BenchmarkHex16(b *testing.B) {
	u := uint16(0xffff)

	for b.Loop() {
		_ = Hex16(u)
	}
}

func BenchmarkHex8(b *testing.B) {
	u := uint8(0xff)

	for b.Loop() {
		_ = Hex8(u)
	}
}
