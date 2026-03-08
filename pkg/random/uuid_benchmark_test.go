package random

import (
	"testing"
)

func BenchmarkUUIDv7(b *testing.B) {
	r := New(nil)

	for b.Loop() {
		_ = r.UUIDv7()
	}
}
