package enumbitmap

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func testEnum(bits int) (map[string]int, map[int]string) {
	esi := make(map[string]int, bits)
	eis := make(map[int]string, bits)

	i := 1
	for bit := 1; bit <= bits; bit++ {
		s := strconv.FormatUint(uint64(i), 2)
		eis[i] = s
		esi[s] = i
		i = (i << 1)
	}

	return esi, eis
}

func Test_MapStringsToUint8(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		s       []string
		want    uint8
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
			name:    "success - all enum values",
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			esi, _ := testEnum(8)
			got, err := MapStringsToUint8(esi, tt.s)
			require.Equal(t, tt.wantErr, err != nil, "MapStringsToUint8() error = %v, wantErr %v", err, tt.wantErr)
			require.Equal(t, tt.want, got, "MapStringsToUint8() got = %v, want %v", got, tt.want)
		})
	}
}

func Test_MapUint8ToStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		v        uint8
		want     []string
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
			name:    "success - all enum values",
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, eis := testEnum(8)
			if tt.enumFunc != nil {
				eis = tt.enumFunc(eis)
			}
			got, err := MapUint8ToStrings(eis, tt.v)
			require.Equal(t, tt.wantErr, err != nil, "MapUint8ToStrings() error = %v, wantErr %v", err, tt.wantErr)
			require.Equal(t, tt.want, got, "MapUint8ToStrings() got = %v, want %v", got, tt.want)
		})
	}
}
