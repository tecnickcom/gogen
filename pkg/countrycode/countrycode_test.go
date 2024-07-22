package countrycode

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_countryByAlpha2ID(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      uint16
		exp     *CountryData
		wantErr bool
	}{
		{
			name: "valid",
			in:   0x0357,
			exp: &CountryData{
				StatusCode:             1,
				Status:                 "Officially assigned",
				Alpha2Code:             "ZW",
				Alpha3Code:             "ZWE",
				NumericCode:            "716",
				NameEnglish:            "Zimbabwe",
				NameFrench:             "Zimbabwe (le)",
				Region:                 "Africa",
				SubRegion:              "Sub-Saharan Africa",
				IntermediateRegion:     "Eastern Africa",
				RegionCode:             "002",
				SubRegionCode:          "202",
				IntermediateRegionCode: "014",
				TLD:                    "zw",
			},
			wantErr: false,
		},
		{
			name:    "invalid key",
			in:      0,
			exp:     nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.countryByAlpha2ID(tt.in)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_countryByAlpha2ID_errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  uint64
	}{
		{
			name: "invalid status",
			key:  0x7D5F572D98181357, // 0 | 111 | 11010 10111 | 11010 10111 00101 | 1011001100 | 00001 | 10000 | 00100 | 11010 10111
		},
		{
			name: "invalid alpha2",
			key:  0x1FFF572D98181357, // 0 | 001 | 11111 11111 | 11010 10111 00101 | 1011001100 | 00001 | 10000 | 00100 | 11010 10111
		},
		{
			name: "invalid region",
			key:  0x1D5F572D99F81357, // 0 | 001 | 11010 10111 | 11010 10111 00101 | 1011001100 | 11111 | 10000 | 00100 | 11010 10111
		},
		{
			name: "invalid sub-region",
			key:  0x1D5F572D981F9357, // 0 | 001 | 11010 10111 | 11010 10111 00101 | 1011001100 | 00001 | 11111 | 00100 | 11010 10111
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := New(nil)

			require.NoError(t, err)
			require.NotNil(t, data)

			data.dCountryKeyByAlpha2ID[1] = tt.key

			got, err := data.countryByAlpha2ID(1)

			require.Error(t, err)
			require.Nil(t, got)
		})
	}
}

func Test_EnumStatus(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)
	got := data.EnumStatus()

	require.Len(t, got, len(data.dStatusByID))
}

func Test_EnumRegion(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)
	got := data.EnumRegion()

	require.Len(t, got, len(data.dRegionByID))
}

func Test_EnumSubRegion(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)
	got := data.EnumSubRegion()

	require.Len(t, got, len(data.dSubRegionByID))
}

func Test_EnumIntermediateRegion(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)
	got := data.EnumIntermediateRegion()

	require.Len(t, got, len(data.dIntermediateRegionByID))
}

func Test_CountryByAlpha2Code(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      string
		exp     *CountryData
		wantErr bool
	}{
		{
			name: "valid",
			in:   "ZW",
			exp: &CountryData{
				StatusCode:             1,
				Status:                 "Officially assigned",
				Alpha2Code:             "ZW",
				Alpha3Code:             "ZWE",
				NumericCode:            "716",
				NameEnglish:            "Zimbabwe",
				NameFrench:             "Zimbabwe (le)",
				Region:                 "Africa",
				SubRegion:              "Sub-Saharan Africa",
				IntermediateRegion:     "Eastern Africa",
				RegionCode:             "002",
				SubRegionCode:          "202",
				IntermediateRegionCode: "014",
				TLD:                    "zw",
			},
			wantErr: false,
		},
		{
			name:    "invalid key",
			in:      "",
			exp:     nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.CountryByAlpha2Code(tt.in)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_CountryByAlpha3Code(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      string
		exp     *CountryData
		wantErr bool
	}{
		{
			name: "valid",
			in:   "ZWE",
			exp: &CountryData{
				StatusCode:             1,
				Status:                 "Officially assigned",
				Alpha2Code:             "ZW",
				Alpha3Code:             "ZWE",
				NumericCode:            "716",
				NameEnglish:            "Zimbabwe",
				NameFrench:             "Zimbabwe (le)",
				Region:                 "Africa",
				SubRegion:              "Sub-Saharan Africa",
				IntermediateRegion:     "Eastern Africa",
				RegionCode:             "002",
				SubRegionCode:          "202",
				IntermediateRegionCode: "014",
				TLD:                    "zw",
			},
			wantErr: false,
		},
		{
			name:    "invalid key",
			in:      "",
			exp:     nil,
			wantErr: true,
		},
		{
			name:    "wrong key",
			in:      "AAA",
			exp:     nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.CountryByAlpha3Code(tt.in)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_CountryByNumericCode(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      string
		exp     *CountryData
		wantErr bool
	}{
		{
			name: "valid",
			in:   "716",
			exp: &CountryData{
				StatusCode:             1,
				Status:                 "Officially assigned",
				Alpha2Code:             "ZW",
				Alpha3Code:             "ZWE",
				NumericCode:            "716",
				NameEnglish:            "Zimbabwe",
				NameFrench:             "Zimbabwe (le)",
				Region:                 "Africa",
				SubRegion:              "Sub-Saharan Africa",
				IntermediateRegion:     "Eastern Africa",
				RegionCode:             "002",
				SubRegionCode:          "202",
				IntermediateRegionCode: "014",
				TLD:                    "zw",
			},
			wantErr: false,
		},
		{
			name:    "invalid key",
			in:      "",
			exp:     nil,
			wantErr: true,
		},
		{
			name:    "wrong key",
			in:      "999",
			exp:     nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.CountryByNumericCode(tt.in)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_countriesByAlpha2IDs(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      []uint16
		exp     int
		wantErr bool
	}{
		{
			name:    "empty",
			in:      []uint16{},
			exp:     0,
			wantErr: false,
		},
		{
			name:    "valid",
			in:      []uint16{0x0357, 0x0358},
			exp:     2,
			wantErr: false,
		},
		{
			name:    "invalid",
			in:      []uint16{0x0357, 0x0000},
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.countriesByAlpha2IDs(tt.in)

			require.Len(t, got, tt.exp)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_countriesByRegionID(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      uint8
		exp     int
		wantErr bool
	}{
		{
			name:    "first",
			in:      0,
			exp:     429,
			wantErr: false,
		},
		{
			name:    "last",
			in:      uint8(len(data.dRegionByID) - 1),
			exp:     51,
			wantErr: false,
		},
		{
			name:    "invalid",
			in:      uint8(len(data.dRegionByID)),
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.countriesByRegionID(tt.in)

			require.Len(t, got, tt.exp)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_CountriesByRegionCode(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      string
		exp     int
		wantErr bool
	}{
		{
			name:    "first",
			in:      "",
			exp:     429,
			wantErr: false,
		},
		{
			name:    "last",
			in:      "150",
			exp:     51,
			wantErr: false,
		},
		{
			name:    "invalid",
			in:      "999",
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.CountriesByRegionCode(tt.in)

			require.Len(t, got, tt.exp)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_CountriesByRegionName(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      string
		exp     int
		wantErr bool
	}{
		{
			name:    "first",
			in:      "",
			exp:     429,
			wantErr: false,
		},
		{
			name:    "last",
			in:      "Europe",
			exp:     51,
			wantErr: false,
		},
		{
			name:    "invalid",
			in:      "INVALID",
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.CountriesByRegionName(tt.in)

			require.Len(t, got, tt.exp)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_countriesBySubRegionID(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      uint8
		exp     int
		wantErr bool
	}{
		{
			name:    "first",
			in:      0,
			exp:     429,
			wantErr: false,
		},
		{
			name:    "last",
			in:      uint8(len(data.dSubRegionByID) - 1),
			exp:     52,
			wantErr: false,
		},
		{
			name:    "invalid",
			in:      uint8(len(data.dSubRegionByID)),
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.countriesBySubRegionID(tt.in)

			require.Len(t, got, tt.exp)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_CountriesBySubRegionCode(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      string
		exp     int
		wantErr bool
	}{
		{
			name:    "first",
			in:      "",
			exp:     429,
			wantErr: false,
		},
		{
			name:    "last",
			in:      "419",
			exp:     52,
			wantErr: false,
		},
		{
			name:    "invalid",
			in:      "INVALID",
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.CountriesBySubRegionCode(tt.in)

			require.Len(t, got, tt.exp)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_CountriesBySubRegionName(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      string
		exp     int
		wantErr bool
	}{
		{
			name:    "first",
			in:      "",
			exp:     429,
			wantErr: false,
		},
		{
			name:    "last",
			in:      "Latin America and the Caribbean",
			exp:     52,
			wantErr: false,
		},
		{
			name:    "invalid",
			in:      "INVALID",
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.CountriesBySubRegionName(tt.in)

			require.Len(t, got, tt.exp)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_countriesByIntermediateRegionID(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      uint8
		exp     int
		wantErr bool
	}{
		{
			name:    "first",
			in:      0,
			exp:     571,
			wantErr: false,
		},
		{
			name:    "last",
			in:      uint8(len(data.dIntermediateRegionByID) - 1),
			exp:     28,
			wantErr: false,
		},
		{
			name:    "invalid",
			in:      uint8(len(data.dIntermediateRegionByID)),
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.countriesByIntermediateRegionID(tt.in)

			require.Len(t, got, tt.exp)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_CountriesByIntermediateRegionCode(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      string
		exp     int
		wantErr bool
	}{
		{
			name:    "first",
			in:      "",
			exp:     571,
			wantErr: false,
		},
		{
			name:    "last",
			in:      "029",
			exp:     28,
			wantErr: false,
		},
		{
			name:    "invalid",
			in:      "999",
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.CountriesByIntermediateRegionCode(tt.in)

			require.Len(t, got, tt.exp)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_CountriesByIntermediateRegionName(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      string
		exp     int
		wantErr bool
	}{
		{
			name:    "first",
			in:      "",
			exp:     571,
			wantErr: false,
		},
		{
			name:    "last",
			in:      "Caribbean",
			exp:     28,
			wantErr: false,
		},
		{
			name:    "invalid",
			in:      "INVALID",
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.CountriesByIntermediateRegionName(tt.in)

			require.Len(t, got, tt.exp)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_CountriesByStatusID(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      uint8
		exp     int
		wantErr bool
	}{
		{
			name:    "first",
			in:      0,
			exp:     321,
			wantErr: false,
		},
		{
			name:    "last",
			in:      uint8(len(data.dStatusByID) - 1),
			exp:     14,
			wantErr: false,
		},
		{
			name:    "official",
			in:      1,
			exp:     249,
			wantErr: false,
		},
		{
			name:    "invalid",
			in:      uint8(len(data.dStatusByID)),
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.CountriesByStatusID(tt.in)

			require.Len(t, got, tt.exp)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_CountriesByStatusName(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      string
		exp     int
		wantErr bool
	}{
		{
			name:    "first",
			in:      "Unassigned",
			exp:     321,
			wantErr: false,
		},
		{
			name:    "last",
			in:      "Formerly assigned",
			exp:     14,
			wantErr: false,
		},
		{
			name:    "official",
			in:      "Officially assigned",
			exp:     249,
			wantErr: false,
		},
		{
			name:    "invalid",
			in:      "",
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.CountriesByStatusName(tt.in)

			require.Len(t, got, tt.exp)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_CountriesByTLD(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      string
		exp     int
		wantErr bool
	}{
		{
			name:    "it",
			in:      "it",
			exp:     1,
			wantErr: false,
		},
		{
			name:    "zw",
			in:      "zw",
			exp:     1,
			wantErr: false,
		},
		{
			name:    "us",
			in:      "us",
			exp:     2,
			wantErr: false,
		},
		{
			name:    "invalid zero",
			in:      "",
			exp:     0,
			wantErr: true,
		},
		{
			name:    "invalid max",
			in:      "ZZZ",
			exp:     0,
			wantErr: true,
		},
		{
			name:    "wrong tld",
			in:      "zz",
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.CountriesByTLD(tt.in)

			require.Len(t, got, tt.exp)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_countryKey(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      *CountryData
		exp     uint64
		expA2   uint16
		wantErr bool
	}{
		{
			name: "valid",
			in: &CountryData{ // 0x0357
				Status:                 "Officially assigned",
				Alpha2Code:             "ZW",
				Alpha3Code:             "ZWE",
				NumericCode:            "716",
				NameEnglish:            "Zimbabwe",
				NameFrench:             "Zimbabwe (le)",
				Region:                 "Africa",
				SubRegion:              "Sub-Saharan Africa",
				IntermediateRegion:     "Eastern Africa",
				RegionCode:             "002",
				SubRegionCode:          "202",
				IntermediateRegionCode: "014",
				TLD:                    "zw",
			},
			exp:     0x1D5F572D98181357,
			expA2:   0x0357,
			wantErr: false,
		},
		{
			name: "error - Status",
			in: &CountryData{
				Status:                 "INVALID",
				Alpha2Code:             "ZW",
				Alpha3Code:             "ZWE",
				NumericCode:            "716",
				NameEnglish:            "Zimbabwe",
				NameFrench:             "Zimbabwe (le)",
				Region:                 "Africa",
				SubRegion:              "Sub-Saharan Africa",
				IntermediateRegion:     "Eastern Africa",
				RegionCode:             "002",
				SubRegionCode:          "202",
				IntermediateRegionCode: "014",
				TLD:                    "zw",
			},
			exp:     0,
			expA2:   0,
			wantErr: true,
		},
		{
			name: "error - Alpha2Code",
			in: &CountryData{
				Status:                 "Officially assigned",
				Alpha2Code:             "",
				Alpha3Code:             "ZWE",
				NumericCode:            "716",
				NameEnglish:            "Zimbabwe",
				NameFrench:             "Zimbabwe (le)",
				Region:                 "Africa",
				SubRegion:              "Sub-Saharan Africa",
				IntermediateRegion:     "Eastern Africa",
				RegionCode:             "002",
				SubRegionCode:          "202",
				IntermediateRegionCode: "014",
				TLD:                    "zw",
			},
			exp:     0,
			expA2:   0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a2, ck, err := data.countryKey(tt.in)

			require.Equal(t, tt.exp, ck)
			require.Equal(t, tt.expA2, a2)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
