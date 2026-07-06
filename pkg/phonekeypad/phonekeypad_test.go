package phonekeypad

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKeypadDigit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		r          rune
		want       int
		wantStatus bool
	}{
		{
			name:       "number",
			r:          '0',
			want:       0,
			wantStatus: true,
		},
		{
			name:       "uppercase letter",
			r:          'S',
			want:       7,
			wantStatus: true,
		},
		{
			name:       "lowercase letter",
			r:          's',
			want:       7,
			wantStatus: true,
		},
		{
			name:       "digit nine",
			r:          '9',
			want:       9,
			wantStatus: true,
		},
		{
			name:       "lowercase z",
			r:          'z',
			want:       9,
			wantStatus: true,
		},
		{
			name:       "invalid",
			r:          '!',
			want:       -1,
			wantStatus: false,
		},
		{
			name:       "accented letter is skipped",
			r:          'é',
			want:       -1,
			wantStatus: false,
		},
		{
			name:       "full-width digit is skipped",
			r:          '２', // U+FF12
			want:       -1,
			wantStatus: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, status := KeypadDigit(tt.r)

			require.Equal(t, tt.want, got)
			require.Equal(t, tt.wantStatus, status)
		})
	}
}

func TestKeypadNumber(t *testing.T) {
	t.Parallel()

	num := "0123456789-ABCDEFGHIJKLMNOPQRSTUVWXYZ-abcdefghijklmnopqrstuvwxyz"
	exp := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 2, 2, 2, 3, 3, 3, 4, 4, 4, 5, 5, 5, 6, 6, 6, 7, 7, 7, 7, 8, 8, 8, 9, 9, 9, 9, 2, 2, 2, 3, 3, 3, 4, 4, 4, 5, 5, 5, 6, 6, 6, 7, 7, 7, 7, 8, 8, 8, 9, 9, 9, 9}

	seq := KeypadNumber(num)

	require.Equal(t, exp, seq)
	require.Len(t, seq, 10+26+26)
}

func TestKeypadNumberEdgeCases(t *testing.T) {
	t.Parallel()

	// Empty input returns a non-nil, empty slice (never nil).
	got := KeypadNumber("")
	require.NotNil(t, got)
	require.Empty(t, got)

	// All-separator input also returns non-nil and empty.
	got = KeypadNumber("-.()/ +#*")
	require.NotNil(t, got)
	require.Empty(t, got)

	// Non-ASCII letters are skipped; only ASCII maps.
	require.Equal(t, []int{2, 2, 3}, KeypadNumber("CAFÉ")) // É dropped
}

func TestKeypadNumberString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		num  string
		exp  string
	}{
		{
			name: "full alphabet and digits",
			num:  "0123456789-ABCDEFGHIJKLMNOPQRSTUVWXYZ-abcdefghijklmnopqrstuvwxyz",
			exp:  "01234567892223334445556667777888999922233344455566677778889999",
		},
		{
			name: "vanity number from godoc example",
			num:  "1-800-FLOWERS",
			exp:  "18003569377",
		},
		{
			name: "empty input",
			num:  "",
			exp:  "",
		},
		{
			name: "all punctuation input",
			num:  "-.()/ +#*",
			exp:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			seq := KeypadNumberString(tt.num)

			require.Equal(t, tt.exp, seq)
		})
	}
}
