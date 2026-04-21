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

func BenchmarkHex64UB(b *testing.B) {
	u := uint64(0xffffffffffffffff)
	dst := [16]byte{}

	for b.Loop() {
		Hex64UB(u, &dst)
	}
}

func BenchmarkHex64B(b *testing.B) {
	src := [8]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

	for b.Loop() {
		_ = Hex64B(src)
	}
}

func BenchmarkHex64BB(b *testing.B) {
	src := [8]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	dst := [16]byte{}

	for b.Loop() {
		Hex64BB(src, &dst)
	}
}

func BenchmarkHex32(b *testing.B) {
	u := uint32(0xffffffff)

	for b.Loop() {
		_ = Hex32(u)
	}
}

func BenchmarkHex32UB(b *testing.B) {
	u := uint32(0xffffffff)
	dst := [8]byte{}

	for b.Loop() {
		Hex32UB(u, &dst)
	}
}

func BenchmarkHex32B(b *testing.B) {
	src := [4]byte{0xff, 0xff, 0xff, 0xff}

	for b.Loop() {
		_ = Hex32B(src)
	}
}

func BenchmarkHex32BB(b *testing.B) {
	src := [4]byte{0xff, 0xff, 0xff, 0xff}
	dst := [8]byte{}

	for b.Loop() {
		Hex32BB(src, &dst)
	}
}

func BenchmarkHex16(b *testing.B) {
	u := uint16(0xffff)

	for b.Loop() {
		_ = Hex16(u)
	}
}

func BenchmarkHex16UB(b *testing.B) {
	u := uint16(0xffff)
	dst := [4]byte{}

	for b.Loop() {
		Hex16UB(u, &dst)
	}
}

func BenchmarkHex16B(b *testing.B) {
	src := [2]byte{0xff, 0xff}

	for b.Loop() {
		_ = Hex16B(src)
	}
}

func BenchmarkHex16BB(b *testing.B) {
	src := [2]byte{0xff, 0xff}
	dst := [4]byte{}

	for b.Loop() {
		Hex16BB(src, &dst)
	}
}

func BenchmarkHex8(b *testing.B) {
	u := uint8(0xff)

	for b.Loop() {
		_ = Hex8(u)
	}
}

func BenchmarkHex8UB(b *testing.B) {
	u := uint8(0xff)
	dst := [2]byte{}

	for b.Loop() {
		Hex8UB(u, &dst)
	}
}

func BenchmarkHex8B(b *testing.B) {
	src := [1]byte{0xff}

	for b.Loop() {
		_ = Hex8B(src)
	}
}

func BenchmarkHex8BB(b *testing.B) {
	src := [1]byte{0xff}
	dst := [2]byte{}

	for b.Loop() {
		Hex8BB(src, &dst)
	}
}
