package random

import (
	"testing"
)

func BenchmarkRnd_RandomBytes(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		sinkBytes, errSink = r.RandomBytes(16)
	}
}

func BenchmarkRnd_RandUint32(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		sinkUint32 = r.RandUint32()
	}
}

func BenchmarkRnd_RandUint64(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		sinkUint64 = r.RandUint64()
	}
}

func BenchmarkRnd_RandHex64(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		sinkString = r.RandHex64()
	}
}

func BenchmarkRnd_RandString64(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		sinkString = r.RandString64()
	}
}

func BenchmarkRnd_RandString(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		sinkString, errSink = r.RandString(16)
	}
}
