package countrycode

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_New(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	require.Len(t, data.dStatusByID, len(data.dStatusByID))
	require.Len(t, data.dStatusIDByName, len(data.dStatusByID))
	require.Len(t, data.dRegionByID, len(data.dRegionByID))
	require.Len(t, data.dRegionIDByCode, len(data.dRegionByID))
	require.Len(t, data.dRegionIDByName, len(data.dRegionByID))
	require.Len(t, data.dSubRegionByID, len(data.dSubRegionByID))
	require.Len(t, data.dSubRegionIDByCode, len(data.dSubRegionByID))
	require.Len(t, data.dSubRegionIDByName, len(data.dSubRegionByID))
	require.Len(t, data.dIntermediateRegionByID, len(data.dIntermediateRegionByID))
	require.Len(t, data.dIntermediateRegionIDByCode, len(data.dIntermediateRegionByID))
	require.Len(t, data.dIntermediateRegionIDByName, len(data.dIntermediateRegionByID))
	require.Len(t, data.dCountryNamesByAlpha2ID, 249)
	require.Len(t, data.dCountryKeyByAlpha2ID, 676)
	require.Len(t, data.dAlpha2IDByAlpha3ID, 249)
	require.Len(t, data.dAlpha2IDByNumericID, 249)
	require.Len(t, data.dAlpha2IDsByRegionID, len(data.dRegionByID))
	require.Len(t, data.dAlpha2IDsBySubRegionID, len(data.dSubRegionByID))
	require.Len(t, data.dAlpha2IDsByIntermediateRegionID, len(data.dIntermediateRegionByID))
	require.Len(t, data.dAlpha2IDsByStatusID, len(data.dStatusByID))
	require.Len(t, data.dAlpha2IDsByTLD, 250)
}

func Test_New_custom_data(t *testing.T) {
	t.Parallel()

	cdata := []*CountryData{
		{
			Status:                 "Officially assigned",
			Alpha2Code:             "ZM",
			Alpha3Code:             "ZMB",
			NumericCode:            "894",
			NameEnglish:            "Zambia",
			NameFrench:             "Zambie (la)",
			Region:                 "Africa",
			SubRegion:              "Sub-Saharan Africa",
			IntermediateRegion:     "Eastern Africa",
			RegionCode:             "002",
			SubRegionCode:          "202",
			IntermediateRegionCode: "014",
			TLD:                    "zm",
		},
		{
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
	}

	data, err := New(cdata)

	require.NoError(t, err)
	require.NotNil(t, data)

	require.Len(t, data.dStatusByID, len(data.dStatusByID))
	require.Len(t, data.dStatusIDByName, len(data.dStatusByID))
	require.Len(t, data.dRegionByID, len(data.dRegionByID))
	require.Len(t, data.dRegionIDByCode, len(data.dRegionByID))
	require.Len(t, data.dRegionIDByName, len(data.dRegionByID))
	require.Len(t, data.dSubRegionByID, len(data.dSubRegionByID))
	require.Len(t, data.dSubRegionIDByCode, len(data.dSubRegionByID))
	require.Len(t, data.dSubRegionIDByName, len(data.dSubRegionByID))
	require.Len(t, data.dIntermediateRegionByID, len(data.dIntermediateRegionByID))
	require.Len(t, data.dIntermediateRegionIDByCode, len(data.dIntermediateRegionByID))
	require.Len(t, data.dIntermediateRegionIDByName, len(data.dIntermediateRegionByID))
	require.Len(t, data.dCountryNamesByAlpha2ID, 2)
	require.Len(t, data.dCountryKeyByAlpha2ID, 2)
	require.Len(t, data.dAlpha2IDByAlpha3ID, 2)
	require.Len(t, data.dAlpha2IDByNumericID, 2)
	require.Len(t, data.dAlpha2IDsByRegionID, 1)
	require.Len(t, data.dAlpha2IDsBySubRegionID, 1)
	require.Len(t, data.dAlpha2IDsByIntermediateRegionID, 1)
	require.Len(t, data.dAlpha2IDsByStatusID, 1)
	require.Len(t, data.dAlpha2IDsByTLD, 2)
}

func Test_New_custom_data_error(t *testing.T) {
	t.Parallel()

	cdata := []*CountryData{
		{
			Status:     "INVALID",
			Alpha2Code: "--",
		},
	}

	data, err := New(cdata)

	require.Error(t, err)
	require.Nil(t, data)
}

func Test_statusByID(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      int
		exp     *enumData
		wantErr bool
	}{
		{
			name:    "first",
			in:      0,
			exp:     &enumData{code: "0", name: "Unassigned"},
			wantErr: false,
		},
		{
			name:    "last",
			in:      len(data.dStatusByID) - 1,
			exp:     &enumData{code: "6", name: "Formerly assigned"},
			wantErr: false,
		},
		{
			name:    "out of range too small",
			in:      -1,
			exp:     nil,
			wantErr: true,
		},
		{
			name:    "out of range too big",
			in:      len(data.dStatusByID),
			exp:     nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.statusByID(tt.in)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_statusIDByName(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      string
		exp     uint8
		wantErr bool
	}{
		{
			name:    "first",
			in:      "Unassigned",
			exp:     0,
			wantErr: false,
		},
		{
			name:    "last",
			in:      "Formerly assigned",
			exp:     uint8(len(data.dStatusByID) - 1),
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

			got, err := data.statusIDByName(tt.in)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_regionByID(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      int
		exp     *enumData
		wantErr bool
	}{
		{
			name:    "first",
			in:      0,
			exp:     &enumData{code: "", name: ""},
			wantErr: false,
		},
		{
			name:    "last",
			in:      len(data.dRegionByID) - 1,
			exp:     &enumData{code: "150", name: "Europe"},
			wantErr: false,
		},
		{
			name:    "out of range too small",
			in:      -1,
			exp:     nil,
			wantErr: true,
		},
		{
			name:    "out of range too big",
			in:      len(data.dRegionByID),
			exp:     nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.regionByID(tt.in)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_regionIDByCode(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      string
		exp     uint8
		wantErr bool
	}{
		{
			name:    "first",
			in:      "",
			exp:     0,
			wantErr: false,
		},
		{
			name:    "last",
			in:      "150",
			exp:     uint8(len(data.dRegionByID) - 1),
			wantErr: false,
		},
		{
			name:    "invalid",
			in:      "invalid",
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.regionIDByCode(tt.in)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_regionIDByName(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      string
		exp     uint8
		wantErr bool
	}{
		{
			name:    "first",
			in:      "",
			exp:     0,
			wantErr: false,
		},
		{
			name:    "last",
			in:      "Europe",
			exp:     uint8(len(data.dRegionByID) - 1),
			wantErr: false,
		},
		{
			name:    "invalid",
			in:      "invalid",
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.regionIDByName(tt.in)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_subRegionByID(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      int
		exp     *enumData
		wantErr bool
	}{
		{
			name:    "first",
			in:      0,
			exp:     &enumData{code: "", name: ""},
			wantErr: false,
		},
		{
			name:    "last",
			in:      len(data.dSubRegionByID) - 1,
			exp:     &enumData{code: "419", name: "Latin America and the Caribbean"},
			wantErr: false,
		},
		{
			name:    "out of range too small",
			in:      -1,
			exp:     nil,
			wantErr: true,
		},
		{
			name:    "out of range too big",
			in:      len(data.dSubRegionByID),
			exp:     nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.subRegionByID(tt.in)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_subRegionIDByCode(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      string
		exp     uint8
		wantErr bool
	}{
		{
			name:    "first",
			in:      "",
			exp:     0,
			wantErr: false,
		},
		{
			name:    "last",
			in:      "419",
			exp:     uint8(len(data.dSubRegionByID) - 1),
			wantErr: false,
		},
		{
			name:    "invalid",
			in:      "invalid",
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.subRegionIDByCode(tt.in)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_subRegionIDByName(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      string
		exp     uint8
		wantErr bool
	}{
		{
			name:    "first",
			in:      "",
			exp:     0,
			wantErr: false,
		},
		{
			name:    "last",
			in:      "Latin America and the Caribbean",
			exp:     uint8(len(data.dSubRegionByID) - 1),
			wantErr: false,
		},
		{
			name:    "invalid",
			in:      "invalid",
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.subRegionIDByName(tt.in)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_intermediateRegionByID(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      int
		exp     *enumData
		wantErr bool
	}{
		{
			name:    "first",
			in:      0,
			exp:     &enumData{code: "", name: ""},
			wantErr: false,
		},
		{
			name:    "last",
			in:      len(data.dIntermediateRegionByID) - 1,
			exp:     &enumData{code: "029", name: "Caribbean"},
			wantErr: false,
		},
		{
			name:    "out of range too small",
			in:      -1,
			exp:     nil,
			wantErr: true,
		},
		{
			name:    "out of range too big",
			in:      len(data.dIntermediateRegionByID),
			exp:     nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.intermediateRegionByID(tt.in)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_intermediateRegionIDByCode(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      string
		exp     uint8
		wantErr bool
	}{
		{
			name:    "first",
			in:      "",
			exp:     0,
			wantErr: false,
		},
		{
			name:    "last",
			in:      "029",
			exp:     uint8(len(data.dIntermediateRegionByID) - 1),
			wantErr: false,
		},
		{
			name:    "invalid",
			in:      "invalid",
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.intermediateRegionIDByCode(tt.in)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_intermediateRegionIDByName(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      string
		exp     uint8
		wantErr bool
	}{
		{
			name:    "first",
			in:      "",
			exp:     0,
			wantErr: false,
		},
		{
			name:    "last",
			in:      "Caribbean",
			exp:     uint8(len(data.dIntermediateRegionByID) - 1),
			wantErr: false,
		},
		{
			name:    "invalid",
			in:      "invalid",
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.intermediateRegionIDByName(tt.in)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_countryNamesByAlpha2ID(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      uint16
		exp     *Names
		wantErr bool
	}{
		{
			name: "first",
			in:   0x0024,
			exp: &Names{
				EN: "Andorra",
				FR: "Andorre (l')",
			},
			wantErr: false,
		},
		{
			name: "last",
			in:   0x0357,
			exp: &Names{
				EN: "Zimbabwe",
				FR: "Zimbabwe (le)",
			},
			wantErr: false,
		},
		{
			name:    "invalid zero",
			in:      0,
			exp:     nil,
			wantErr: true,
		},
		{
			name:    "invalid max",
			in:      0xFFFF,
			exp:     nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.countryNamesByAlpha2ID(tt.in)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_countryKeyByAlpha2ID(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      uint16
		exp     uint64
		wantErr bool
	}{
		{
			name:    "first",
			in:      0x0021,
			exp:     0x2084000000000000,
			wantErr: false,
		},
		{
			name:    "last",
			in:      0x035A,
			exp:     0x2D68000000000000,
			wantErr: false,
		},
		{
			name:    "first official",
			in:      0x0024,
			exp:     0x10902E2028530024,
			wantErr: false,
		},
		{
			name:    "last official",
			in:      0x0357,
			exp:     0x1D5F572D98181357,
			wantErr: false,
		},
		{
			name:    "invalid zero",
			in:      0,
			exp:     0,
			wantErr: true,
		},
		{
			name:    "invalid max",
			in:      0xFFFF,
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.countryKeyByAlpha2ID(tt.in)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_alpha2IDByAlpha3ID(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      uint16
		exp     uint16
		wantErr bool
	}{
		{
			name:    "first",
			in:      0x05C4,
			exp:     0x0024,
			wantErr: false,
		},
		{
			name:    "last",
			in:      0x6AE5,
			exp:     0x0357,
			wantErr: false,
		},
		{
			name:    "invalid zero",
			in:      0,
			exp:     0,
			wantErr: true,
		},
		{
			name:    "invalid max",
			in:      0xFFFF,
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.alpha2IDByAlpha3ID(tt.in)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_alpha2IDByNumericID(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      uint16
		exp     uint16
		wantErr bool
	}{
		{
			name:    "first",
			in:      0x0014,
			exp:     0x0024,
			wantErr: false,
		},
		{
			name:    "last",
			in:      0x02CC,
			exp:     0x0357,
			wantErr: false,
		},
		{
			name:    "invalid zero",
			in:      0,
			exp:     0,
			wantErr: true,
		},
		{
			name:    "invalid max",
			in:      0xFFFF,
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.alpha2IDByNumericID(tt.in)

			require.Equal(t, tt.exp, got)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_alpha2IDsByRegionID(t *testing.T) {
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

			got, err := data.alpha2IDsByRegionID(tt.in)

			require.Len(t, got, tt.exp)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_alpha2IDsBySubRegionID(t *testing.T) {
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

			got, err := data.alpha2IDsBySubRegionID(tt.in)

			require.Len(t, got, tt.exp)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_alpha2IDsByIntermediateRegionID(t *testing.T) {
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

			got, err := data.alpha2IDsByIntermediateRegionID(tt.in)

			require.Len(t, got, tt.exp)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_alpha2IDsByStatusID(t *testing.T) {
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

			got, err := data.alpha2IDsByStatusID(tt.in)

			require.Len(t, got, tt.exp)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func Test_alpha2IDsByTLD(t *testing.T) {
	t.Parallel()

	data, err := New(nil)

	require.NoError(t, err)
	require.NotNil(t, data)

	tests := []struct {
		name    string
		in      uint16
		exp     int
		wantErr bool
	}{
		{
			name:    "first",
			in:      0x0024,
			exp:     1,
			wantErr: false,
		},
		{
			name:    "last",
			in:      0x0357,
			exp:     1,
			wantErr: false,
		},
		{
			name:    "usa",
			in:      0x02B3,
			exp:     2,
			wantErr: false,
		},
		{
			name:    "zero",
			in:      0,
			exp:     426,
			wantErr: false,
		},
		{
			name:    "invalid max",
			in:      0xFFFF,
			exp:     0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.alpha2IDsByTLD(tt.in)

			require.Len(t, got, tt.exp)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
