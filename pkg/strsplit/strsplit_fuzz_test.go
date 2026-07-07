package strsplit

import (
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

// dropSpace removes every Unicode whitespace rune so two strings can be compared
// on their non-whitespace content alone.
func dropSpace(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}

		return r
	}, s)
}

// assertChunkInvariants verifies the properties shared by the output of both
// [Chunk] and [ChunkLine] for the given input.
func assertChunkInvariants(t *testing.T, s string, out []string, size, n int) {
	t.Helper()

	// invariants that hold for any input, including arbitrary (invalid) bytes
	for _, c := range out {
		require.NotEmpty(t, c, "chunk must never be empty")
		require.Equal(t, c, strings.TrimSpace(c), "chunk must be whitespace-trimmed")
	}

	if n > 0 {
		require.LessOrEqual(t, len(out), n, "produced more chunks than n")
	}

	// The remaining invariants assume well-formed UTF-8 input; Go fuzzing also
	// feeds arbitrary bytes, for which the splitter deliberately preserves the
	// raw (possibly invalid) input rather than corrupting it further.
	if !utf8.ValidString(s) {
		return
	}

	for _, c := range out {
		require.True(t, utf8.ValidString(c), "chunk %q is not valid UTF-8", c)

		// only a single wide rune may exceed size; multi-rune chunks must fit
		if utf8.RuneCountInString(c) > 1 {
			require.LessOrEqual(t, len(c), size, "multi-rune chunk exceeds size")
		}
	}

	// unlimited output over a valid size must preserve every non-whitespace rune
	if (size >= 1) && (n < 0) {
		require.Equal(t, dropSpace(s), dropSpace(strings.Join(out, "")),
			"non-whitespace content changed")
	}
}

func FuzzChunkLine(f *testing.F) {
	seeds := []struct {
		s    string
		size int
		n    int
	}{
		{"", 5, -1},
		{"a b c", 2, -1},
		{"😀😀", 3, -1},
		{"hello,world", 8, -1},
		{"   x   ", 2, -1},
		{"a\r\nb", 4, 2},
		{"Hello 世界 Hello", 7, 3}, //nolint:gosmopolitan
	}
	for _, s := range seeds {
		f.Add(s.s, s.size, s.n)
	}

	f.Fuzz(func(t *testing.T, s string, size, n int) {
		assertChunkInvariants(t, s, ChunkLine(s, size, n), size, n)
	})
}

func FuzzChunk(f *testing.F) {
	seeds := []struct {
		s    string
		size int
		n    int
	}{
		{"", 5, -1},
		{"hello\nworld", 5, -1},
		{"a\n\n\nb", 3, -1},
		{"line one\nlong-second-line-here", 6, 4},
		{"😀\n😀😀", 3, 2},
	}
	for _, s := range seeds {
		f.Add(s.s, s.size, s.n)
	}

	f.Fuzz(func(t *testing.T, s string, size, n int) {
		assertChunkInvariants(t, s, Chunk(s, size, n), size, n)
	})
}
