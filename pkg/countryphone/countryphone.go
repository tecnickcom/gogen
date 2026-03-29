/*
Package countryphone resolves international phone number prefixes into
country and regional metadata.

It addresses a common backend problem: mapping a dialed number (or prefix) to
structured geographic and number-type information that can be used for
validation, formatting, routing, fraud controls, and analytics.

The package models country calling code behavior defined by the International
Telecommunication Union (ITU), including the ITU-T E.123 and E.164 numbering
recommendations.

# How it works

countryphone indexes prefix data in a trie and performs longest-prefix lookup.
This gives fast reads and deterministic matches for variable-length numbering
plans. The default dataset is embedded, and callers can provide custom data
through New when they need provider-specific or updated plans.

Top features

  - Longest-prefix matching for accurate geographic resolution across mixed
    prefix lengths.
  - Rich metadata output via NumInfo and GeoInfo, including country code,
    area/group name, and typed classifications.
  - Stable enum helpers (NumberType and AreaType) that convert integer types
    into readable labels.
  - Custom dataset injection using InData, InCountryData, and InPrefixGroup,
    so teams can adapt to private numbering rules without forking package code.

Why this matters

  - Consistent prefix resolution across services reduces subtle routing and
    validation bugs.
  - Trie-based lookup is efficient for both single-request APIs and bulk
    processing pipelines.
  - The data model composes naturally with country metadata from
    github.com/tecnickcom/gogen/pkg/countrycode.

# Typical usage

Create a resolver with New, then call NumberInfo for a prefix or full number.
When no prefix matches, NumberInfo returns an error.
*/
package countryphone

import (
	"fmt"

	"github.com/tecnickcom/gogen/pkg/numtrie"
)

// InPrefixGroup stores the type and geographical information of a group of phone
// number prefixes.
type InPrefixGroup struct {
	// Name is the name of the group or geographical area.
	Name string `json:"name"`

	// Type is the type of group or geographical area:
	//   - 0 = ""
	//   - 1 = "state"
	//   - 2 = "province or territory"
	//   - 3 = "nation or territory"
	//   - 4 = "non-geographic"
	//   - 5 = "other"
	Type int `json:"type"`

	// PrefixType is the type of phone number prefix:
	//   - 0 = ""
	//   - 1 = "landline"
	//   - 2 = "mobile"
	//   - 3 = "pager"
	//   - 4 = "satellite"
	//   - 5 = "special service"
	//   - 6 = "virtual"
	//   - 7 = "other"
	PrefixType int `json:"prefixType"`

	// Prefixes is a list of phone number prefixes (including the Country Code).
	Prefixes []string `json:"prefixes"`
}

// InCountryData stores all the phone number prefixes information for a country.
type InCountryData struct {
	// CC is the Country Calling code (e.g. "1" for "US" and "CA").
	CC string `json:"cc"`

	// Groups is a list of phone prefixes information grouped by geographical
	// area or type.
	Groups []InPrefixGroup `json:"groups"`
}

// InData maps country alpha-2 codes to their dialing metadata.
type InData = map[string]*InCountryData

// GeoInfo stores geographical information of a phone number.
type GeoInfo struct {
	// Alpha2 is the ISO-3166 Alpha-2 Country Code.
	Alpha2 string `json:"alpha2"`

	// Area is the geographical area.
	Area string `json:"area"`

	// Type is the type of area:
	//   - 0 = ""
	//   - 1 = "state"
	//   - 2 = "province or territory"
	//   - 3 = "nation or territory"
	//   - 4 = "non-geographic"
	//   - 5 = "other"
	Type int `json:"type"`
}

// NumInfo stores the number type and geographical information of a phone number.
type NumInfo struct {
	// Type is the type of number:
	//   - 0 = ""
	//   - 1 = "landline"
	//   - 2 = "mobile"
	//   - 3 = "pager"
	//   - 4 = "satellite"
	//   - 5 = "special service"
	//   - 6 = "virtual"
	//   - 7 = "other"
	Type int `json:"type"`

	// Geo is the geographical information.
	Geo []*GeoInfo `json:"geo"`
}

// PrefixData maps normalized dialing prefixes to number metadata.
type PrefixData = map[string]*NumInfo

// Data stores numbering metadata and prefix indexes used for longest-prefix lookups.
type Data struct {
	enumNumberType [8]string
	enumAreaType   [6]string
	trie           *numtrie.Node[NumInfo]
}

// New builds a prefix resolver backed by a longest-prefix trie.
//
// If data is nil, the embedded default numbering dataset is loaded.
// Precomputing trie indexes at construction keeps NumberInfo lookups fast and
// deterministic for mixed-length international prefixes.
func New(data InData) *Data {
	d := &Data{}

	d.loadEnums()

	if data == nil {
		data = defaultData()
	}

	d.loadData(data)

	return d
}

// NumberInfo resolves a phone number or prefix into typed geographic metadata.
//
// The lookup uses longest-prefix matching, so callers can pass either full
// numbers or partial prefixes and still obtain the most specific available
// mapping.
//
// NOTE: see the "github.com/tecnickcom/gogen/pkg/countrycode" package to get the
// country information from the Alpha2 code.
func (d *Data) NumberInfo(num string) (*NumInfo, error) {
	data, status := d.trie.Get(num)

	if status < 0 || data == nil {
		return nil, fmt.Errorf("no match for prefix %s", num)
	}

	return data, nil
}

// NumberType returns the label for a numeric prefix-type code.
//
// It converts compact integer values into readable type names for APIs,
// logging, and analytics outputs.
func (d *Data) NumberType(t int) (string, error) {
	if t < 0 || t >= len(d.enumNumberType) {
		return "", fmt.Errorf("invalid number type %d", t)
	}

	return d.enumNumberType[t], nil
}

// AreaType returns the label for a numeric area-type code.
//
// It translates stored integer codes into stable human-readable category names.
func (d *Data) AreaType(t int) (string, error) {
	if t < 0 || t >= len(d.enumAreaType) {
		return "", fmt.Errorf("invalid area type %d", t)
	}

	return d.enumAreaType[t], nil
}

// loadEnums initializes canonical labels for number and area type codes.
func (d *Data) loadEnums() {
	d.enumNumberType = [...]string{
		"",
		"landline",
		"mobile",
		"pager",
		"satellite",
		"special service",
		"virtual",
		"other",
	}

	d.enumAreaType = [...]string{
		"",
		"state",
		"province or territory",
		"nation or territory",
		"non-geographic",
		"other",
	}
}

// insertPrefix inserts or merges prefix metadata into the trie.
//
// When a prefix already exists, geographic entries are merged so multiple
// countries/areas can be represented for shared numbering spaces.
func (d *Data) insertPrefix(prefix string, info *NumInfo) {
	v, status := d.trie.Get(prefix)

	if (status == numtrie.StatusMatchFull || status == numtrie.StatusMatchPartial) &&
		(v != nil) && (len(v.Geo) > 0) {
		// the node already exists > merge the data
		if len(info.Geo) > 0 {
			info.Geo = append(v.Geo, info.Geo...)
		}
	}

	d.trie.Add(prefix, info)
}

// insertGroups expands country group definitions into trie prefixes.
//
// Each group becomes one or more prefix entries that carry number type and
// geographic annotations.
func (d *Data) insertGroups(a2 string, cdata *InCountryData) {
	for _, g := range cdata.Groups {
		groupInfo := &NumInfo{
			Type: g.PrefixType,
			Geo: []*GeoInfo{
				{
					Alpha2: a2,
					Area:   g.Name,
					Type:   g.Type,
				},
			},
		}

		if len(g.Prefixes) == 0 {
			d.insertPrefix(cdata.CC, groupInfo)
			continue
		}

		for _, p := range g.Prefixes {
			d.insertPrefix(p, groupInfo)
		}
	}
}

// loadData constructs trie nodes from the input numbering dataset.
//
// It inserts one root country-calling-code entry per country and then appends
// optional group-level prefixes for more specific matches.
func (d *Data) loadData(data InData) {
	d.trie = numtrie.New[NumInfo]()

	doneRootCC := make(map[string]bool, (26*26)+1) // all possible CCs + 1 for non-geographic

	for a2, cdata := range data {
		if _, ok := doneRootCC[a2]; !ok {
			// insert the root node for the country code only once
			d.insertPrefix(cdata.CC, &NumInfo{
				Type: 0,
				Geo: []*GeoInfo{
					{
						Alpha2: a2,
						Area:   "",
						Type:   0,
					},
				},
			})

			doneRootCC[a2] = true
		}

		if len(cdata.Groups) > 0 {
			d.insertGroups(a2, cdata)
		}
	}
}
