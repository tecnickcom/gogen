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
			name:       "invalid",
			r:          '!',
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

func TestKeypadNumberString(t *testing.T) {
	t.Parallel()

	num := "0123456789-ABCDEFGHIJKLMNOPQRSTUVWXYZ-abcdefghijklmnopqrstuvwxyz"
	exp := "01234567892223334445556667777888999922233344455566677778889999"

	seq := KeypadNumberString(num)

	require.Equal(t, exp, seq)
}
