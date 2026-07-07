package strsplit

import (
	"strings"
	"testing"
)

// Package-level sink keeps the splitter output observable so the compiler cannot
// dead-code-eliminate the work being measured.
//
//nolint:gochecknoglobals // benchmark sink that defeats dead-code elimination
var sink []string

func BenchmarkChunkLineASCII(b *testing.B) {
	s := strings.Repeat("the quick brown fox jumps over ", 40)

	for b.Loop() {
		sink = ChunkLine(s, 40, -1)
	}
}

func BenchmarkChunkLineNoSeparators(b *testing.B) {
	s := strings.Repeat("a", 2000)

	for b.Loop() {
		sink = ChunkLine(s, 40, -1)
	}
}

func BenchmarkChunkLinePunctuation(b *testing.B) {
	s := strings.Repeat("alpha,beta;gamma.delta:", 40)

	for b.Loop() {
		sink = ChunkLine(s, 40, -1)
	}
}

func BenchmarkChunkLineUnicode(b *testing.B) {
	s := strings.Repeat("héllo 世界 😀 wörld ", 40) //nolint:gosmopolitan

	for b.Loop() {
		sink = ChunkLine(s, 40, -1)
	}
}

func BenchmarkChunkMultiline(b *testing.B) {
	line := strings.Repeat("the quick brown fox ", 5)
	s := strings.Repeat(line+"\n", 40)

	for b.Loop() {
		sink = Chunk(s, 40, -1)
	}
}
