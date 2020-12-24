// +build benchmark

package uidc

import (
	"sync"
	"testing"
)

func BenchmarkNewID64(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewID64()
	}
}

func BenchmarkNewID128(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewID128()
	}
}
