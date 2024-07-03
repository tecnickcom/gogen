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
			args: []string{"学院路30号", " ăâîșț  ĂÂÎȘȚ  "},
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
