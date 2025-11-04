package uidc

import (
	"testing"
)

func BenchmarkNewID64(b *testing.B) {
	for b.Loop() {
		_ = NewID64()
	}
}

func BenchmarkNewID128(b *testing.B) {
	for b.Loop() {
		_ = NewID128()
	}
}
