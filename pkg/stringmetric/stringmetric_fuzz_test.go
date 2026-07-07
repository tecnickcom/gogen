package stringmetric

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

// FuzzDLDistance exercises the metric invariants that [DLDistance] must uphold
// for every input: non-negativity, identity of indiscernibles, symmetry, the
// max(len) upper bound, and the triangle inequality (which the unrestricted
// Damerau-Levenshtein metric satisfies but the OSA variant does not).
func FuzzDLDistance(f *testing.F) {
	seeds := []struct {
		a, b, c string
	}{
		{"", "", ""},
		{"a", "", "ab"},
		{"AB", "BA", "AA"},
		{"a cat", "a act", "a abct"},
		{"aba", "bab", "aab"},
		{"CA", "ABC", "AC"},
		{"αβγδ", "αδ", "βγ"},
		{"INTENTION", "EXECUTION", "CONVENTION"},
	}
	for _, s := range seeds {
		f.Add(s.a, s.b, s.c)
	}

	f.Fuzz(func(t *testing.T, a, b, c string) {
		dab := DLDistance(a, b)

		// non-negativity
		require.GreaterOrEqual(t, dab, 0, "distance must be non-negative")

		// identity of indiscernibles: d(x, x) == 0
		require.Equal(t, 0, DLDistance(a, a), "distance to self must be 0")

		// symmetry: d(a, b) == d(b, a)
		require.Equal(t, dab, DLDistance(b, a), "distance must be symmetric")

		// upper bound: d(a, b) <= max(len(a), len(b)) in runes
		require.LessOrEqual(t, dab,
			max(utf8.RuneCountInString(a), utf8.RuneCountInString(b)),
			"distance must not exceed the longer rune length")

		// triangle inequality: d(a, c) <= d(a, b) + d(b, c)
		require.LessOrEqual(t, DLDistance(a, c), dab+DLDistance(b, c),
			"triangle inequality must hold")
	})
}
