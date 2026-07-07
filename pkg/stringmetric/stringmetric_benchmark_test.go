package stringmetric

import (
	"strings"
	"testing"
)

func BenchmarkDLDistance(b *testing.B) {
	benchmarks := []struct {
		name string
		sa   string
		sb   string
	}{
		{"equal", "intention", "intention"},                                      // sa == sb fast path
		{"empty", "intention", ""},                                               // one empty string
		{"short_ascii", "intention", "execution"},                                // typical short input
		{"medium_ascii", strings.Repeat("abcde", 8), strings.Repeat("abfde", 8)}, // 40x40 grid
		{"unicode", "αβγδεζηθικ", "αβγδεζηθικλ"},                                 // multi-byte runes
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for b.Loop() {
				_ = DLDistance(bm.sa, bm.sb)
			}
		})
	}
}
