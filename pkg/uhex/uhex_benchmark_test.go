package uhex

import (
	"testing"
)

// Package-level sinks keep encoder output observable so the compiler cannot
// dead-code-eliminate the work being measured. Inputs are mutated every
// iteration to defeat loop-invariant hoisting and constant folding.
//
//nolint:gochecknoglobals // benchmark sinks that defeat dead-code elimination
var (
	sink2     [2]byte
	sink4     [4]byte
	sink8     [8]byte
	sink16    [16]byte
	sinkSlice []byte
)

func BenchmarkHex64(b *testing.B) {
	var u uint64

	for b.Loop() {
		u++
		sink16 = [16]byte(Hex64(u))
	}
}

func BenchmarkHex64UB(b *testing.B) {
	var u uint64

	for b.Loop() {
		u++
		Hex64UB(u, &sink16)
	}
}

func BenchmarkHex64B(b *testing.B) {
	var src [8]byte

	for b.Loop() {
		src[0]++
		sink16 = [16]byte(Hex64B(src))
	}
}

func BenchmarkHex64BB(b *testing.B) {
	var src [8]byte

	for b.Loop() {
		src[0]++
		Hex64BB(src, &sink16)
	}
}

func BenchmarkHex32(b *testing.B) {
	var u uint32

	for b.Loop() {
		u++
		sink8 = [8]byte(Hex32(u))
	}
}

func BenchmarkHex32UB(b *testing.B) {
	var u uint32

	for b.Loop() {
		u++
		Hex32UB(u, &sink8)
	}
}

func BenchmarkHex32B(b *testing.B) {
	var src [4]byte

	for b.Loop() {
		src[0]++
		sink8 = [8]byte(Hex32B(src))
	}
}

func BenchmarkHex32BB(b *testing.B) {
	var src [4]byte

	for b.Loop() {
		src[0]++
		Hex32BB(src, &sink8)
	}
}

func BenchmarkHex16(b *testing.B) {
	var u uint16

	for b.Loop() {
		u++
		sink4 = [4]byte(Hex16(u))
	}
}

func BenchmarkHex16UB(b *testing.B) {
	var u uint16

	for b.Loop() {
		u++
		Hex16UB(u, &sink4)
	}
}

func BenchmarkHex16B(b *testing.B) {
	var src [2]byte

	for b.Loop() {
		src[0]++
		sink4 = [4]byte(Hex16B(src))
	}
}

func BenchmarkHex16BB(b *testing.B) {
	var src [2]byte

	for b.Loop() {
		src[0]++
		Hex16BB(src, &sink4)
	}
}

func BenchmarkHex8(b *testing.B) {
	var u uint8

	for b.Loop() {
		u++
		sink2 = [2]byte(Hex8(u))
	}
}

func BenchmarkHex8UB(b *testing.B) {
	var u uint8

	for b.Loop() {
		u++
		Hex8UB(u, &sink2)
	}
}

func BenchmarkHex8B(b *testing.B) {
	var src [1]byte

	for b.Loop() {
		src[0]++
		sink2 = [2]byte(Hex8B(src))
	}
}

func BenchmarkHex8BB(b *testing.B) {
	var src [1]byte

	for b.Loop() {
		src[0]++
		Hex8BB(src, &sink2)
	}
}

// BenchmarkHex64Escaping measures the allocation path of the slice-returning
// API: the result escapes to a package-level sink, forcing the backing array
// onto the heap (one allocation per call).
func BenchmarkHex64Escaping(b *testing.B) {
	var u uint64

	for b.Loop() {
		u++
		sinkSlice = Hex64(u)
	}
}

// BenchmarkHex64BEscaping is the byte-array counterpart of BenchmarkHex64Escaping.
func BenchmarkHex64BEscaping(b *testing.B) {
	var src [8]byte

	for b.Loop() {
		src[0]++
		sinkSlice = Hex64B(src)
	}
}
