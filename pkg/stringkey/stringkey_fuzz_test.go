package stringkey

import (
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"
)

// doubleEveryWhitespace duplicates every Unicode whitespace rune. Because
// consecutive whitespace collapses to a single space and leading/trailing
// whitespace is trimmed, this must never change the key.
func doubleEveryWhitespace(s string) string {
	var b strings.Builder

	for _, r := range s {
		b.WriteRune(r)

		if unicode.IsSpace(r) {
			b.WriteRune(r)
		}
	}

	return b.String()
}

// wrapWhitespace surrounds s with assorted Unicode whitespace that trimming
// must remove, leaving the key unchanged.
func wrapWhitespace(s string) string {
	return " \t " + s + " 　\n"
}

// FuzzNew exercises the canonicalization and representation invariants that
// [New] must uphold for every input: determinism, insensitivity to case and to
// surrounding/duplicated whitespace, and the contract of the three output
// representations.
func FuzzNew(f *testing.F) {
	seeds := []struct {
		a, b, c string
	}{
		{"", "", ""},
		{"a", "b", "c"},
		{"0123456789", "abcdefghijklmnopqrstuvwxyz", "Lorem ipsum dolor sit amet"},
		{"学院路30号", " ăâîșț  ĂÂÎȘȚ  ", "MiXeD CaSe"}, //nolint:gosmopolitan
		{"Å", "Å", "café"}, // composed vs decomposed
		{"  hi  ", "a\tb", "line\nbreak"},
		{"İstanbul", "ẞtraße", "　x　y　"},
	}
	for _, s := range seeds {
		f.Add(s.a, s.b, s.c)
	}

	f.Fuzz(func(t *testing.T, a, b, c string) {
		key := New(a, b, c).Key()

		// determinism: the same input always yields the same key
		require.Equal(t, key, New(a, b, c).Key(), "New must be deterministic")

		// case insensitivity: lowercasing the input first changes nothing,
		// because New already lowercases (and unicode.ToLower is idempotent)
		require.Equal(t, key,
			New(strings.ToLower(a), strings.ToLower(b), strings.ToLower(c)).Key(),
			"key must be case-insensitive")

		// whitespace insensitivity: duplicating every whitespace rune collapses
		// back to the same normalized bytes
		require.Equal(t, key,
			New(doubleEveryWhitespace(a), doubleEveryWhitespace(b), doubleEveryWhitespace(c)).Key(),
			"duplicated whitespace must collapse")

		// whitespace insensitivity: surrounding whitespace is trimmed away
		require.Equal(t, key,
			New(wrapWhitespace(a), wrapWhitespace(b), wrapWhitespace(c)).Key(),
			"surrounding whitespace must be trimmed")

		// representation contract: Hex is a 16-char lowercase hex encoding of the
		// key, and String is its base-36 encoding; both must round-trip
		sk := New(a, b, c)

		hex := sk.Hex()
		require.Len(t, hex, 16, "Hex must be 16 characters")

		hv, err := strconv.ParseUint(hex, 16, 64)
		require.NoError(t, err, "Hex must be valid hexadecimal")
		require.Equal(t, key, hv, "Hex must encode the key")

		sv, err := strconv.ParseUint(sk.String(), 36, 64)
		require.NoError(t, err, "String must be valid base-36")
		require.Equal(t, key, sv, "String must encode the key")
	})
}
