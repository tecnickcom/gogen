package countrycode

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_decodeCountryKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input uint64
		exp   *countryKeyElem
	}{
		{
			name:  "zero",
			input: 0x0000000000000000,
			exp:   &countryKeyElem{},
		},
		{
			name:  "max",
			input: 0xFFFFFFFFFFFFFFFF,
			exp: &countryKeyElem{
				status:    0x07,
				alpha2:    0x3FF,
				alpha3:    0x7FFF,
				numeric:   0x3FF,
				region:    0x1F,
				subregion: 0x1F,
				intregion: 0x1F,
				tld:       0x3FF,
			},
		},
		{ // 0x0165
			name:  "Kenya",
			input: 0x1595657328181165, // 0 | 001 | 01011 00101 | 01011 00101 01110 | 0110010100 | 00001 | 10000 | 00100 | 01011 00101
			exp: &countryKeyElem{
				status:    0x01,   // 001 = "Officially assigned"
				alpha2:    0x0165, // 01011 00101 => [11,5] => 11+64=75=K 5+64=69=E =>"KE"
				alpha3:    0x2CAE, // 01011 00101 01110 => [11,5,14] => 11+64=75=K 5+64=69=E 14+64=78=N => "KEN"
				numeric:   0x0194, // 0110010100 => "404"
				region:    0x01,   // 00001 => 1 => {"002", "Africa"}
				subregion: 0x10,   // 10000 => 16 = {"202", "Sub-Saharan Africa"}
				intregion: 0x04,   // 00100 => 4 => {"014", "Eastern Africa"}
				tld:       0x0165, // 01011 00101 => [11,5] => 11+96=107=k 5+96=101=e => "ke"
			},
		},
		{ // 0x0268
			name:  "Saint Helena, Ascension and Tristan da Cunha",
			input: 0x19A268751C180A68, // 0 | 001 | 10011 01000 | 10011 01000 01110 | 1010001110 | 00001 | 10000 | 00010 | 10011 01000
			exp: &countryKeyElem{
				status:    0x01,   // 1 = "Officially assigned"
				alpha2:    0x0268, // 10011 01000 => [19,8] => 19+64=83=S 8+64=72=H => "SH"
				alpha3:    0x4D0E, // 10011 01000 01110 => [19,8,14] => 19+64=83=S 8+64=72=H 14+64=78=N => "SHN"
				numeric:   0x028E, // 1010001110 => "654"
				region:    0x01,   // 00001 => 1 => {"002", "Africa"}
				subregion: 0x10,   // 10000 => 16 => {"202", "Sub-Saharan Africa"}
				intregion: 0x02,   // 00010 => 2 => {"011", "Western Africa"}
				tld:       0x0268, // 10011 01000 => [19,8] => 19+96=115=s 8+96=104=h => "sh"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := decodeCountryKey(tt.input)

			require.Equal(t, tt.exp, got)
		})
	}
}

func Test_encodeCountryKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		elem *countryKeyElem
		exp  uint64
	}{
		{
			name: "zero",
			elem: &countryKeyElem{},
			exp:  0x0000000000000000,
		},
		{
			name: "max",
			elem: &countryKeyElem{
				status:    0x07,
				alpha2:    0x3FF,
				alpha3:    0x7FFF,
				numeric:   0x3FF,
				region:    0x1F,
				subregion: 0x1F,
				intregion: 0x1F,
				tld:       0x3FF,
			},
			exp: 0x7FFFFFFFFFFFFFFF,
		},
		{
			name: "Kenya",
			elem: &countryKeyElem{
				status:    0x01,   // 001 = "Officially assigned"
				alpha2:    0x0165, // 01011 00101 => [11,5] => 11+64=75=K 5+64=69=E =>"KE"
				alpha3:    0x2CAE, // 01011 00101 01110 => [11,5,14] => 11+64=75=K 5+64=69=E 14+64=78=N => "KEN"
				numeric:   0x0194, // 0110010100 => "404"
				region:    0x01,   // 00001 => 1 => {"002", "Africa"}
				subregion: 0x10,   // 10000 => 16 = {"202", "Sub-Saharan Africa"}
				intregion: 0x04,   // 00100 => 4 => {"014", "Eastern Africa"}
				tld:       0x0165, // 01011 00101 => [11,5] => 11+96=107=k 5+96=101=e => "ke"
			},
			exp: 0x1595657328181165, // 0 | 001 | 01011 00101 | 01011 00101 01110 | 0110010100 | 00001 | 10000 | 00100 | 01011 00101
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.elem.encodeCountryKey()

			require.Equal(t, tt.exp, got)
		})
	}
}

func Test_charOffsetUpper(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   byte
		exp     uint16
		wantErr bool
	}{
		{
			name:    "valid first",
			input:   'A',
			exp:     0x01,
			wantErr: false,
		},
		{
			name:    "valid last",
			input:   'Z',
			exp:     0x1a,
			wantErr: false,
		},
		{
			name:    "invalid char",
			input:   '1',
			exp:     0,
			wantErr: true,
		},
		{
			name:    "invalid case",
			input:   'a',
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := charOffsetUpper(tt.input)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_charOffsetLower(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   byte
		exp     uint16
		wantErr bool
	}{
		{
			name:    "valid first",
			input:   'a',
			exp:     0x01,
			wantErr: false,
		},
		{
			name:    "valid last",
			input:   'z',
			exp:     0x1a,
			wantErr: false,
		},
		{
			name:    "invalid char",
			input:   '1',
			exp:     0,
			wantErr: true,
		},
		{
			name:    "invalid case",
			input:   'A',
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := charOffsetLower(tt.input)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_encodeAlpha2(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		exp     uint16
		wantErr bool
	}{
		{
			name:    "valid",
			input:   "IT",
			exp:     0x0134, // 00 01001 10100 => [9,20] => 9+64=73=I, 20+64=84=T
			wantErr: false,
		},
		{
			name:    "long",
			input:   "ITALY",
			exp:     0,
			wantErr: true,
		},
		{
			name:    "short",
			input:   "I",
			exp:     0,
			wantErr: true,
		},
		{
			name:    "invalid first",
			input:   "1A",
			exp:     0,
			wantErr: true,
		},
		{
			name:    "invalid second",
			input:   "A1",
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := encodeAlpha2(tt.input)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_decodeAlpha2(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input uint16
		exp   string
	}{
		{
			name:  "valid",
			input: 0x0134, // 00 01001 10100 => [9,20] => 9+64=73=I, 20+64=84=T
			exp:   "IT",
		},
		{
			name:  "invalid zero",
			input: 0,
			exp:   "@@",
		},
		{
			name:  "invalid max",
			input: 0xFFFF,
			exp:   "__",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := decodeAlpha2(tt.input)

			require.Equal(t, tt.exp, got)
		})
	}
}

func Test_encodeAlpha3(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		exp     uint16
		wantErr bool
	}{
		{
			name:    "valid",
			input:   "ITA",
			exp:     0x2681, // 0 01001 10100 00001 => [9,20,1] => 9+64=73=I, 20+64=84=T, 1+64=65=A
			wantErr: false,
		},
		{
			name:    "long",
			input:   "ITALY",
			exp:     0,
			wantErr: true,
		},
		{
			name:    "short",
			input:   "IT",
			exp:     0,
			wantErr: true,
		},
		{
			name:    "invalid",
			input:   "17",
			exp:     0,
			wantErr: true,
		},
		{
			name:    "invalid first",
			input:   "1AA",
			exp:     0,
			wantErr: true,
		},
		{
			name:    "invalid second",
			input:   "A1A",
			exp:     0,
			wantErr: true,
		},
		{
			name:    "invalid third",
			input:   "AA1",
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := encodeAlpha3(tt.input)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_decodeAlpha3(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input uint16
		exp   string
	}{
		{
			name:  "valid",
			input: 0x2681, // 0 01001 10100 00001 => [9,20,1] => 9+64=73=I, 20+64=84=T, 1+64=65=A
			exp:   "ITA",
		},
		{
			name:  "invalid zero",
			input: 0,
			exp:   "@@@",
		},
		{
			name:  "invalid max",
			input: 0xFFFF,
			exp:   "___",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := decodeAlpha3(tt.input)

			require.Equal(t, tt.exp, got)
		})
	}
}

func Test_encodeTLD(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		exp     uint16
		wantErr bool
	}{
		{
			name:    "valid",
			input:   "it",
			exp:     0x0134, // 00 01001 10100 => [9,20] => 9+96=105=i, 20+96=116=t
			wantErr: false,
		},
		{
			name:    "long",
			input:   "ita",
			exp:     0,
			wantErr: true,
		},
		{
			name:    "short",
			input:   "i",
			exp:     0,
			wantErr: true,
		},
		{
			name:    "invalid",
			input:   "17",
			exp:     0,
			wantErr: true,
		},
		{
			name:    "invalid first",
			input:   "1a",
			exp:     0,
			wantErr: true,
		},
		{
			name:    "invalid second",
			input:   "a1",
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := encodeTLD(tt.input)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_decodeTLD(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input uint16
		exp   string
	}{
		{
			name:  "valid",
			input: 0x0134, // 00 01001 10100 => [9,20] => 9+96=105=i, 20+96=116=t
			exp:   "it",
		},
		{
			name:  "invalid zero",
			input: 0,
			exp:   "``",
		},
		{
			name:  "invalid max",
			input: 0xFFFF,
			exp:   "\x7f\x7f",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := decodeTLD(tt.input)

			require.Equal(t, tt.exp, got)
		})
	}
}

func Test_encodeNumeric(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		exp     uint16
		wantErr bool
	}{
		{
			name:    "valid",
			input:   "123",
			exp:     123,
			wantErr: false,
		},
		{
			name:    "long",
			input:   "1234",
			exp:     0,
			wantErr: true,
		},
		{
			name:    "short",
			input:   "12",
			exp:     0,
			wantErr: true,
		},
		{
			name:    "invalid",
			input:   "1a3",
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := encodeNumeric(tt.input)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
