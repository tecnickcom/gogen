package stringkey

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type testItem struct {
	name string
	args []string
	want *StringKey
	key  uint64
	str  string
	hex  string
}

func getTestData() []testItem {
	return []testItem{
		{
			name: "empty set",
			args: []string{},
			want: &StringKey{key: 0x9ae16a3b2f90404f},
			key:  0x9ae16a3b2f90404f,
			str:  "2csgylx78en2n",
			hex:  "9ae16a3b2f90404f",
		},
		{
			name: "empty string",
			args: []string{""},
			want: &StringKey{key: 0x41c0124dcd479182},
			key:  0x41c0124dcd479182,
			str:  "zzuce204aflu",
			hex:  "41c0124dcd479182",
		},
		{
			name: "numbers and letter",
			args: []string{"0123456789", "abcdefghijklmnopqrstuvwxyz", "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."},
			want: &StringKey{key: 0xcacb9eb3194029d6},
			key:  0xcacb9eb3194029d6,
			str:  "330sxpll17r2u",
			hex:  "cacb9eb3194029d6",
		},
		{
			name: "chinese address and romanian diacritics",
			args: []string{"学院路30号", " ăâîșț  ĂÂÎȘȚ  "}, //nolint:gosmopolitan
			want: &StringKey{key: 0xc8bca6255513b74},
			key:  0xc8bca6255513b74,
			str:  "6v9iypdk4l10",
			hex:  "0c8bca6255513b74",
		},
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	for _, tt := range getTestData() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sk := New(tt.args...)

			require.Equal(t, tt.want, sk)
			require.Equal(t, tt.key, sk.Key())
			require.Equal(t, tt.str, sk.String())
			require.Equal(t, tt.hex, sk.Hex())
		})
	}
}

func TestNewComposedEqualsDecomposed(t *testing.T) {
	t.Parallel()

	// Canonically equivalent forms must produce the same key: NFC recomposes the
	// decomposed sequence into the precomposed character before hashing.
	tests := []struct {
		name       string
		precom, de string
	}{
		{
			name:   "A with ring above",
			precom: "Å",  // Å  LATIN CAPITAL LETTER A WITH RING ABOVE
			de:     "Å", // A + COMBINING RING ABOVE
		},
		{
			name:   "e with acute",
			precom: "café",  // café  (precomposed é)
			de:     "café", // cafe + COMBINING ACUTE ACCENT
		},
		{
			name:   "o with diaeresis",
			precom: "ö",  // ö  LATIN SMALL LETTER O WITH DIAERESIS
			de:     "ö", // o + COMBINING DIAERESIS
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, New(tt.precom).Key(), New(tt.de).Key())
		})
	}
}

func TestNewFieldBoundaries(t *testing.T) {
	t.Parallel()

	// The tab separator preserves field boundaries, so the number and grouping
	// of fields is significant: these must all be distinct keys.
	none := New().Key()
	oneEmpty := New("").Key()
	twoEmpty := New("", "").Key()
	split := New("a", "b").Key()
	joined := New("a b").Key()
	tabbed := New("a\tb").Key()

	require.NotEqual(t, none, oneEmpty, "no fields must differ from one empty field")
	require.NotEqual(t, oneEmpty, twoEmpty, "one empty field must differ from two")
	require.NotEqual(t, split, joined, "two fields must differ from one joined field")

	// An embedded tab is Unicode whitespace: it collapses to a single space,
	// so it can never be confused with the field separator.
	require.Equal(t, joined, tabbed, "embedded tab must collapse like a space")
}

func TestNewUnicodeWhitespace(t *testing.T) {
	t.Parallel()

	// regression: interior Unicode whitespace must be collapsed like ASCII
	// whitespace, consistently with the Unicode-aware trimming.
	want := New("a b").Key()

	tests := []struct {
		name string
		arg  string
	}{
		{
			name: "no-break space",
			arg:  "a\u00a0b",
		},
		{
			name: "em space run",
			arg:  "a \u2003 b",
		},
		{
			name: "ideographic space with unicode trim",
			arg:  "\u3000a\u3000b\u3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, want, New(tt.arg).Key())
		})
	}
}
