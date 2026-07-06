package enumbitmap

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func testEnum() (map[string]int, map[int]string) {
	esi := make(map[string]int, maxBit)
	eis := make(map[int]string, maxBit)

	i := 1

	for bit := 1; bit <= maxBit; bit++ {
		s := strconv.FormatInt(int64(i), 2)
		eis[i] = s
		esi[s] = i
		i = (i << 1)
	}

	return esi, eis
}

func Test_BitMapToStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		want     []string
		v        int
		wantErr  bool
		enumFunc func(enum map[int]string) map[int]string
	}{
		{
			name:    "success - empty",
			v:       0,
			want:    []string{},
			wantErr: false,
		},
		{
			name:    "success - one enum value",
			v:       0b00000001,
			want:    []string{"1"},
			wantErr: false,
		},
		{
			name:    "success - two enum values",
			v:       0b00000101,
			want:    []string{"1", "100"},
			wantErr: false,
		},
		{
			name:    "success - MSB and LSB",
			v:       0b10000000_00000000_00000000_00000001,
			want:    []string{"1", "10000000000000000000000000000000"},
			wantErr: false,
		},
		{
			name:    "success - multiple enum values",
			v:       0b11111111,
			want:    []string{"1", "10", "100", "1000", "10000", "100000", "1000000", "10000000"},
			wantErr: false,
		},
		{
			name: "error - invalid bit",
			v:    0b10000000,
			enumFunc: func(enum map[int]string) map[int]string {
				delete(enum, 0b10000000)
				return enum
			},
			want:    []string{},
			wantErr: true,
		},
		{
			name: "error - invalid and valid bit",
			v:    0b10001001,
			enumFunc: func(enum map[int]string) map[int]string {
				delete(enum, 0b10000000)
				return enum
			},
			want:    []string{"1", "1000"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, eis := testEnum()
			if tt.enumFunc != nil {
				eis = tt.enumFunc(eis)
			}

			got, err := BitMapToStrings(eis, tt.v)

			require.Equal(t, tt.wantErr, err != nil, "error = %v, wantErr %v", err, tt.wantErr)
			require.Equal(t, tt.want, got, "got = %v, want %v", got, tt.want)
		})
	}
}

func Test_BitMapToStrings_unknownBitReportsMask(t *testing.T) {
	t.Parallel()

	_, eis := testEnum()
	delete(eis, 0b10000000)

	got, err := BitMapToStrings(eis, 0b10000000)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrUnknownBitValues)
	require.Empty(t, got)
	// The unknown set bit must be reported as its mask value (128), not as a
	// 1-based loop index (8).
	require.Contains(t, err.Error(), "128")
	require.NotContains(t, err.Error(), "[8]")
}

func Test_StringsToBitMap_unknownReportsSentinel(t *testing.T) {
	t.Parallel()

	esi, _ := testEnum()

	got, err := StringsToBitMap(esi, []string{"1", "nope"})
	require.ErrorIs(t, err, ErrUnknownStringValues)
	require.Equal(t, 1, got)
}

func Test_BitMapToStrings_zeroIDUnreachable(t *testing.T) {
	t.Parallel()

	// An ID of 0 can never be produced as a set bit: value 0 always decodes to an
	// empty slice, and no other input references mask 0.
	got, err := BitMapToStrings(map[int]string{0: "none"}, 0)
	require.NoError(t, err)
	require.Equal(t, []string{}, got)
}

func Test_RoundTrip(t *testing.T) {
	t.Parallel()

	esi, eis := testEnum()

	names := []string{"1", "1000", "10000000000000000000000000000000"}

	v, err := StringsToBitMap(esi, names)
	require.NoError(t, err)

	got, err := BitMapToStrings(eis, v)
	require.NoError(t, err)
	require.ElementsMatch(t, names, got)
}

func Test_BitMapToStrings_highBitsIgnored(t *testing.T) {
	t.Parallel()

	_, eis := testEnum()

	// Bit at position 32 (mask 1<<32) is beyond the 32-bit limit and must be
	// silently ignored without producing an error.
	got, err := BitMapToStrings(eis, 1<<32)
	require.NoError(t, err)
	require.Empty(t, got)
}

func Test_StringsToBitMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		s       []string
		want    int
		wantErr bool
	}{
		{
			name:    "success - empty",
			s:       []string{},
			want:    0,
			wantErr: false,
		},
		{
			name:    "success - one enum value",
			s:       []string{"1"},
			want:    0b00000001,
			wantErr: false,
		},
		{
			name:    "success - two enum values",
			s:       []string{"1", "100"},
			want:    0b00000101,
			wantErr: false,
		},
		{
			name:    "success - MSB and LSB",
			s:       []string{"1", "10000000000000000000000000000000"},
			want:    0b10000000_00000000_00000000_00000001,
			wantErr: false,
		},
		{
			name:    "success - multiple enum values",
			s:       []string{"1", "10", "100", "1000", "10000", "100000", "1000000", "10000000"},
			want:    0b11111111,
			wantErr: false,
		},
		{
			name:    "error - invalid enum value",
			s:       []string{"invalid"},
			want:    0,
			wantErr: true,
		},
		{
			name:    "error - invalid and valid enum values",
			s:       []string{"1", "invalid1", "1000", "invalid2"},
			want:    0b00001001,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			esi, _ := testEnum()
			got, err := StringsToBitMap(esi, tt.s)
			require.Equal(t, tt.wantErr, err != nil, "error = %v, wantErr %v", err, tt.wantErr)
			require.Equal(t, tt.want, got, "got = %v, want %v", got, tt.want)
		})
	}
}
