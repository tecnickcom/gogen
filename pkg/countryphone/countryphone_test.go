package countryphone

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	// custom data
	indata := InData{
		"US": &InCountryData{
			CC: "1",
			Groups: []InPrefixGroup{
				{
					Name:       "Alaska",
					Type:       1,
					PrefixType: 1,
					Prefixes:   []string{"1907"},
				},
			},
		},
	}

	data := New(indata)

	require.NotNil(t, data)
}

func TestNew_default(t *testing.T) {
	t.Parallel()

	data := New(nil)

	require.NotNil(t, data)
}

func TestData_NumberInfo(t *testing.T) {
	t.Parallel()

	// load defaut data
	data := New(nil)

	require.NotNil(t, data)

	tests := []struct {
		name    string
		prefix  string
		want    *NumInfo
		wantErr bool
	}{
		{
			name:    "empty",
			prefix:  "",
			want:    nil,
			wantErr: true,
		},
		{
			name:   "non-geographic",
			prefix: "87012345678",
			want: &NumInfo{
				Type: 5,
				Geo: []*GeoInfo{
					{
						Alpha2: "__",
						Area:   "Inmarsat",
						Type:   4,
					},
				},
			},
			wantErr: false,
		},
		{
			name:   "vatican (rome)",
			prefix: "37912345678",
			want: &NumInfo{
				Type: 0,
				Geo: []*GeoInfo{
					{
						Alpha2: "VA",
						Area:   "",
						Type:   0,
					},
				},
			},
			wantErr: false,
		},
		{
			name:   "vatican (unused code)",
			prefix: "39066981234",
			want: &NumInfo{
				Type: 1,
				Geo: []*GeoInfo{
					{
						Alpha2: "VA",
						Area:   "Vatican City",
						Type:   0,
					},
				},
			},
			wantErr: false,
		},
		{
			name:   "california",
			prefix: "1357123456",
			want: &NumInfo{
				Type: 1,
				Geo: []*GeoInfo{
					{
						Alpha2: "US",
						Area:   "California",
						Type:   1,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.NumberInfo(tt.prefix)

			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, got)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want.Type, got.Type)
			require.ElementsMatch(t, tt.want.Geo, got.Geo)
		})
	}
}

func TestData_NumberInfo_custom(t *testing.T) {
	t.Parallel()

	indata := InData{
		"US": &InCountryData{
			CC: "1",
			Groups: []InPrefixGroup{
				{
					Name:       "Alaska",
					Type:       1,
					PrefixType: 1,
					Prefixes:   []string{"1907"},
				},
				{
					Name:       "Arizona",
					Type:       1,
					PrefixType: 1,
					Prefixes:   []string{"1480", "5120", "1602", "1623", "1928"},
				},
			},
		},
		"CA": &InCountryData{
			CC: "1",
			Groups: []InPrefixGroup{
				{
					Name:       "Manitoba",
					Type:       2,
					PrefixType: 1,
					Prefixes:   []string{"1204", "1431", "1584"},
				},
				{
					Name:       "Nunavut",
					Type:       2,
					PrefixType: 1,
					Prefixes:   []string{"1867"},
				},
			},
		},
		"JP": &InCountryData{
			CC: "81",
		},
		"__": &InCountryData{
			CC: "7",
			Groups: []InPrefixGroup{
				{
					Name:       "TEST",
					Type:       5,
					PrefixType: 7,
					Prefixes:   []string{},
				},
			},
		},
	}

	data := New(indata)

	require.NotNil(t, data)

	tests := []struct {
		name    string
		prefix  string
		want    *NumInfo
		wantErr bool
	}{
		{
			name:    "empty",
			prefix:  "",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "no match",
			prefix:  "999999",
			want:    nil,
			wantErr: true,
		},
		{
			name:   "US & CA",
			prefix: "100000",
			want: &NumInfo{
				Type: 0,
				Geo: []*GeoInfo{
					{
						Alpha2: "US",
						Area:   "",
						Type:   0,
					},
					{
						Alpha2: "CA",
						Area:   "",
						Type:   0,
					},
				},
			},
			wantErr: false,
		},
		{
			name:   "US - Alaska",
			prefix: "1907000",
			want: &NumInfo{
				Type: 1,
				Geo: []*GeoInfo{
					{
						Alpha2: "US",
						Area:   "Alaska",
						Type:   1,
					},
				},
			},
			wantErr: false,
		},
		{
			name:   "US - Arizona",
			prefix: "1623000",
			want: &NumInfo{
				Type: 1,
				Geo: []*GeoInfo{
					{
						Alpha2: "US",
						Area:   "Arizona",
						Type:   1,
					},
				},
			},
			wantErr: false,
		},
		{
			name:   "CA - Manitoba",
			prefix: "1431000",
			want: &NumInfo{
				Type: 1,
				Geo: []*GeoInfo{
					{
						Alpha2: "CA",
						Area:   "Manitoba",
						Type:   2,
					},
				},
			},
			wantErr: false,
		},
		{
			name:   "JP",
			prefix: "81234567890",
			want: &NumInfo{
				Type: 0,
				Geo: []*GeoInfo{
					{
						Alpha2: "JP",
						Area:   "",
						Type:   0,
					},
				},
			},
			wantErr: false,
		},
		{
			name:   "Artificial without prefix",
			prefix: "7123",
			want: &NumInfo{
				Type: 7,
				Geo: []*GeoInfo{
					{
						Alpha2: "__",
						Area:   "",
						Type:   0,
					},
					{
						Alpha2: "__",
						Area:   "TEST",
						Type:   5,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.NumberInfo(tt.prefix)

			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, got)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want.Type, got.Type)
			require.ElementsMatch(t, tt.want.Geo, got.Geo)
		})
	}
}

func TestData_NumberInfo_noAncestorGeoMerge(t *testing.T) {
	t.Parallel()

	// The default data contains the "__" entry (empty CC), "FI" (CC "358"),
	// and "AX" (CC "35818"). The regression this covers depended on the map
	// iteration order during New (e.g. "AX" inserted before "FI"), so the
	// resolver is rebuilt multiple times to exercise different orders.
	for range 50 {
		data := New(nil)

		require.NotNil(t, data)

		got, err := data.NumberInfo("3581234567")

		require.NoError(t, err)
		require.Equal(t, 0, got.Type)
		// Only the FI geo data must be returned: no "__" (or other ancestor)
		// entries merged into the more specific prefix.
		require.Len(t, got.Geo, 1)
		require.Equal(t, "FI", got.Geo[0].Alpha2)
	}
}

func TestData_NumberInfo_noMatch(t *testing.T) {
	t.Parallel()

	// load default data
	data := New(nil)

	require.NotNil(t, data)

	// A number matching no stored prefix must return an error instead of the
	// universal "__" fallback previously installed on the trie root.
	got, err := data.NumberInfo("999999999")

	require.Error(t, err)
	require.Nil(t, got)
}

func TestData_NumberInfo_returnsIndependentCopy(t *testing.T) {
	t.Parallel()

	indata := InData{
		"US": &InCountryData{
			CC: "1",
			Groups: []InPrefixGroup{
				{
					Name:       "Arizona",
					Type:       1,
					PrefixType: 1,
					// Two sibling prefixes share the same group definition.
					Prefixes: []string{"1480", "1602"},
				},
			},
		},
	}

	data := New(indata)

	require.NotNil(t, data)

	// First lookup, then corrupt the returned value.
	got1, err := data.NumberInfo("1480000")
	require.NoError(t, err)
	require.Len(t, got1.Geo, 1)

	got1.Type = 99
	got1.Geo[0].Alpha2 = "XX"
	got1.Geo[0].Area = "MUTATED"
	got1.Geo[0].Type = 99
	got1.Geo = append(got1.Geo, &GeoInfo{Alpha2: "ZZ"})

	// A subsequent lookup of the SAME prefix must be unaffected.
	got1again, err := data.NumberInfo("1480000")
	require.NoError(t, err)
	require.Equal(t, 1, got1again.Type)
	require.Len(t, got1again.Geo, 1)
	require.Equal(t, "US", got1again.Geo[0].Alpha2)
	require.Equal(t, "Arizona", got1again.Geo[0].Area)
	require.Equal(t, 1, got1again.Geo[0].Type)

	// A subsequent lookup of a SIBLING prefix must be unaffected.
	got2, err := data.NumberInfo("1602000")
	require.NoError(t, err)
	require.Equal(t, 1, got2.Type)
	require.Len(t, got2.Geo, 1)
	require.Equal(t, "US", got2.Geo[0].Alpha2)
	require.Equal(t, "Arizona", got2.Geo[0].Area)
	require.Equal(t, 1, got2.Geo[0].Type)
}

func TestData_NumberInfo_noDuplicateGeoOnOverlappingPrefixes(t *testing.T) {
	t.Parallel()

	indata := InData{
		"US": &InCountryData{
			CC: "1",
			Groups: []InPrefixGroup{
				{
					Name:       "Arizona",
					Type:       1,
					PrefixType: 1,
					// The same prefix is listed twice within the group.
					Prefixes: []string{"1480", "1480"},
				},
			},
		},
	}

	data := New(indata)

	require.NotNil(t, data)

	got, err := data.NumberInfo("1480000")
	require.NoError(t, err)
	// A duplicated/overlapping prefix must not duplicate GeoInfo entries.
	require.Len(t, got.Geo, 1)
	require.Equal(t, "US", got.Geo[0].Alpha2)
	require.Equal(t, "Arizona", got.Geo[0].Area)
}

func TestData_NumberInfo_nilGeoCopy(t *testing.T) {
	t.Parallel()

	indata := InData{
		"US": &InCountryData{
			CC: "1",
			Groups: []InPrefixGroup{
				{
					Name:       "NoGeo",
					Type:       1,
					PrefixType: 2,
					Prefixes:   []string{"1555"},
				},
			},
		},
	}

	data := New(indata)

	require.NotNil(t, data)

	// Force a node whose stored value has a nil Geo slice to exercise the
	// nil-guard inside the clone path.
	data.trie.Add("1555", &NumInfo{Type: 3, Geo: nil})

	got, err := data.NumberInfo("1555000")
	require.NoError(t, err)
	require.Equal(t, 3, got.Type)
	require.Nil(t, got.Geo)
}

func TestData_NumberType(t *testing.T) {
	t.Parallel()

	data := New(InData{})

	require.NotNil(t, data)

	tests := []struct {
		name    string
		num     int
		want    string
		wantErr bool
	}{
		{
			name:    "out of bounds < 0",
			num:     -1,
			want:    "",
			wantErr: true,
		},
		{
			name:    "out of bounds > max",
			num:     8,
			want:    "",
			wantErr: true,
		},
		{
			name:    "first",
			num:     0,
			want:    "",
			wantErr: false,
		},
		{
			name:    "last",
			num:     7,
			want:    "other",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.NumberType(tt.num)

			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, got)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestData_AreaType(t *testing.T) {
	t.Parallel()

	data := New(InData{})

	require.NotNil(t, data)

	tests := []struct {
		name    string
		num     int
		want    string
		wantErr bool
	}{
		{
			name:    "out of bounds < 0",
			num:     -1,
			want:    "",
			wantErr: true,
		},
		{
			name:    "out of bounds > max",
			num:     6,
			want:    "",
			wantErr: true,
		},
		{
			name:    "first",
			num:     0,
			want:    "",
			wantErr: false,
		},
		{
			name:    "last",
			num:     5,
			want:    "other",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := data.AreaType(tt.num)

			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, got)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
