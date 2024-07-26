/*
Package countryphone provides geographical information of phone numbers. Country
phone codes are defined by the International Telecommunication Union (ITU) in
ITU-T standards E.123 and E.164.

The default phone prefixes data can be overridden with a custom dataset. The
data is stored in a trie for fast prefix matching.
*/
package countryphone

import (
	"fmt"

	"github.com/Vonage/gosrvlib/pkg/numtrie"
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

// InData is a type alias for a map of country Alpha-2 codes to InCountryData.
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

// PrefixData is a type alias for a map of phone number prefixes to NumData.
type PrefixData = map[string]*NumInfo

// Data is the main data structure that stores phone number prefixes and their
// information.
type Data struct {
	enumNumberType [8]string
	enumAreaType   [6]string
	trie           *numtrie.Node[NumInfo]
}

// New initialize the search trie with the given data.
// If data is nil, the embedded default dataset is used.
func New(data InData) *Data {
	d := &Data{}

	d.loadEnums()

	if data == nil {
		data = defaultData()
	}

	d.loadData(data)

	return d
}

// NumberInfo returns the number type and geographical information for the given
// phone number prefix.
//
// NOTE: see the "github.com/Vonage/gosrvlib/pkg/countrycode" package to get the
// country information from the Alpha2 code.
func (d *Data) NumberInfo(num string) (*NumInfo, error) {
	data, status := d.trie.Get(num)

	if status < 0 || data == nil {
		return nil, fmt.Errorf("no match for prefix %s", num)
	}

	return data, nil
}

// NumberType returns the string representation of the number type.
func (d *Data) NumberType(t int) (string, error) {
	if t < 0 || t >= len(d.enumNumberType) {
		return "", fmt.Errorf("invalid number type %d", t)
	}

	return d.enumNumberType[t], nil
}

// AreaType returns the string representation of the area type.
func (d *Data) AreaType(t int) (string, error) {
	if t < 0 || t >= len(d.enumAreaType) {
		return "", fmt.Errorf("invalid area type %d", t)
	}

	return d.enumAreaType[t], nil
}

// loadEnums initializes the enumeration arrays.
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

func (d *Data) insertGroups(a2, cc string, data []InPrefixGroup) {
	if len(data) == 0 {
		return
	}

	for _, g := range data {
		info := &NumInfo{
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
			d.insertPrefix(cc, info)
			continue
		}

		for _, p := range g.Prefixes {
			d.insertPrefix(p, info)
		}
	}
}

// loadData loads the phone number prefixes and their data into the trie.
func (d *Data) loadData(data InData) {
	d.trie = numtrie.New[NumInfo]()

	for k, v := range data {
		info := &NumInfo{
			Type: 0,
			Geo: []*GeoInfo{
				{
					Alpha2: k,
					Area:   "",
					Type:   0,
				},
			},
		}

		d.insertPrefix(v.CC, info)
		d.insertGroups(k, v.CC, v.Groups)
	}
}
