package countrycode

import (
	"sort"
	"strings"
)

// enumData contains the code and name of an enumeration.
type enumData struct {
	code string
	name string
}

// Names contains the English and French Names of a country.
type Names struct {
	EN string
	FR string
}

// Data contains the internal country Data and various indexes.
type Data struct {
	dStatusByID                      [7]*enumData
	dStatusIDByName                  map[string]uint8
	dRegionByID                      []*enumData
	dRegionIDByCode                  map[string]uint8
	dRegionIDByName                  map[string]uint8
	dSubRegionByID                   []*enumData
	dSubRegionIDByCode               map[string]uint8
	dSubRegionIDByName               map[string]uint8
	dIntermediateRegionByID          []*enumData
	dIntermediateRegionIDByCode      map[string]uint8
	dIntermediateRegionIDByName      map[string]uint8
	dCountryNamesByAlpha2ID          map[uint16]*Names
	dCountryKeyByAlpha2ID            map[uint16]uint64
	dAlpha2IDByAlpha3ID              map[uint16]uint16
	dAlpha2IDByNumericID             map[uint16]uint16
	dAlpha2IDsByRegionID             map[uint8][]uint16
	dAlpha2IDsBySubRegionID          map[uint8][]uint16
	dAlpha2IDsByIntermediateRegionID map[uint8][]uint16
	dAlpha2IDsByStatusID             map[uint8][]uint16
	dAlpha2IDsByTLD                  map[uint16][]uint16
}

// New generates the country data, including various indexes.
// If the cdata parameter is nil, the default data is used.
// The generated object should be reused to avoid copying the data.
//
// Default data sources (updated at: 2024-07-17):
//   - https://www.iso.org/iso-3166-country-codes.html
//   - https://www.cia.gov/the-world-factbook/references/country-data-codes/
//   - https://unstats.un.org/unsd/methodology/m49/overview/
//   - https://en.wikipedia.org/wiki/ISO_3166
//   - https://en.wikipedia.org/wiki/ISO_3166-1
//   - https://en.wikipedia.org/wiki/ISO_3166-2
//   - https://en.wikipedia.org/wiki/ISO_3166-3
//   - https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2
//   - https://en.wikipedia.org/wiki/ISO_3166-1_alpha-3
//   - https://en.wikipedia.org/wiki/ISO_3166-1_numeric
func New(cdata []*CountryData) (*Data, error) {
	d := &Data{}

	d.statusMap()

	if cdata == nil {
		d.defaultData()
	} else {
		if err := d.loadData(cdata); err != nil {
			return nil, err
		}
	}

	d.genIndexes()

	return d, nil
}

//nolint:gocognit
func (d *Data) loadData(cdata []*CountryData) error {
	dCountryNamesByAlpha2ID := make(map[uint16]*Names, len(cdata))
	dCountryKeyByAlpha2ID := make(map[uint16]uint64, len(cdata))

	dRegionByID := map[string]*enumData{"": {code: "", name: ""}}
	regionKeys := []string{""}

	dSubRegionByID := map[string]*enumData{"": {code: "", name: ""}}
	subRegionKeys := []string{""}

	dIntermediateRegionByID := map[string]*enumData{"": {code: "", name: ""}}
	intRegionKeys := []string{""}

	for _, country := range cdata {
		a2, ck, err := d.countryKey(country)
		if err != nil {
			return err
		}

		dCountryKeyByAlpha2ID[a2] = ck

		if _, ok := dRegionByID[country.RegionCode]; !ok {
			dRegionByID[country.RegionCode] = &enumData{code: country.RegionCode, name: country.Region}
			regionKeys = append(regionKeys, country.RegionCode)
		}

		if _, ok := dSubRegionByID[country.SubRegionCode]; !ok {
			dSubRegionByID[country.SubRegionCode] = &enumData{code: country.SubRegionCode, name: country.SubRegion}
			subRegionKeys = append(subRegionKeys, country.SubRegionCode)
		}

		if _, ok := dIntermediateRegionByID[country.IntermediateRegionCode]; !ok {
			dIntermediateRegionByID[country.IntermediateRegionCode] = &enumData{code: country.IntermediateRegionCode, name: country.IntermediateRegion}
			intRegionKeys = append(intRegionKeys, country.IntermediateRegionCode)
		}

		if len(country.NameEnglish) > 0 {
			dCountryNamesByAlpha2ID[a2] = &Names{
				EN: country.NameEnglish,
				FR: country.NameFrench,
			}
		}
	}

	d.dCountryNamesByAlpha2ID = dCountryNamesByAlpha2ID
	d.dCountryKeyByAlpha2ID = dCountryKeyByAlpha2ID

	sort.Strings(regionKeys)

	for _, k := range regionKeys {
		d.dRegionByID = append(d.dRegionByID, dRegionByID[k])
	}

	sort.Strings(subRegionKeys)

	for _, k := range subRegionKeys {
		d.dSubRegionByID = append(d.dSubRegionByID, dSubRegionByID[k])
	}

	sort.Strings(intRegionKeys)

	for _, k := range intRegionKeys {
		d.dIntermediateRegionByID = append(d.dIntermediateRegionByID, dIntermediateRegionByID[k])
	}

	return nil
}

func (d *Data) statusMap() {
	d.dStatusByID = [...]*enumData{
		{"0", "Unassigned"},
		{"1", "Officially assigned"},
		{"2", "User-assigned"},
		{"3", "Exceptionally reserved"},
		{"4", "Transitionally reserved"},
		{"5", "Indeterminately reserved"},
		{"6", "Formerly assigned"},
	}

	d.dStatusIDByName = make(map[string]uint8, len(d.dStatusByID))
	for k, v := range d.dStatusByID {
		d.dStatusIDByName[strings.ToUpper(v.name)] = uint8(k)
	}
}

func (d *Data) genIndexes() {
	d.dRegionIDByCode = make(map[string]uint8, len(d.dRegionByID))
	d.dRegionIDByName = make(map[string]uint8, len(d.dRegionByID))

	for k, v := range d.dRegionByID {
		d.dRegionIDByCode[v.code] = uint8(k)
		d.dRegionIDByName[strings.ToUpper(v.name)] = uint8(k)
	}

	d.dSubRegionIDByCode = make(map[string]uint8, len(d.dSubRegionByID))
	d.dSubRegionIDByName = make(map[string]uint8, len(d.dSubRegionByID))

	for k, v := range d.dSubRegionByID {
		d.dSubRegionIDByCode[v.code] = uint8(k)
		d.dSubRegionIDByName[strings.ToUpper(v.name)] = uint8(k)
	}

	d.dIntermediateRegionIDByCode = make(map[string]uint8, len(d.dIntermediateRegionByID))
	d.dIntermediateRegionIDByName = make(map[string]uint8, len(d.dIntermediateRegionByID))

	for k, v := range d.dIntermediateRegionByID {
		d.dIntermediateRegionIDByCode[v.code] = uint8(k)
		d.dIntermediateRegionIDByName[strings.ToUpper(v.name)] = uint8(k)
	}

	// extra indexes

	d.dAlpha2IDByAlpha3ID = make(map[uint16]uint16, len(d.dCountryKeyByAlpha2ID))
	d.dAlpha2IDByNumericID = make(map[uint16]uint16, len(d.dCountryKeyByAlpha2ID))
	d.dAlpha2IDsByRegionID = make(map[uint8][]uint16, len(d.dRegionByID))
	d.dAlpha2IDsBySubRegionID = make(map[uint8][]uint16, len(d.dSubRegionByID))
	d.dAlpha2IDsByIntermediateRegionID = make(map[uint8][]uint16, len(d.dIntermediateRegionByID))
	d.dAlpha2IDsByStatusID = make(map[uint8][]uint16, len(d.dStatusByID))
	d.dAlpha2IDsByTLD = make(map[uint16][]uint16, len(d.dCountryKeyByAlpha2ID))

	for k, v := range d.dCountryKeyByAlpha2ID {
		ck := decodeCountryKey(v)

		d.dAlpha2IDByAlpha3ID[ck.alpha3] = k
		d.dAlpha2IDByNumericID[ck.numeric] = k
		d.dAlpha2IDsByRegionID[ck.region] = append(d.dAlpha2IDsByRegionID[ck.region], k)
		d.dAlpha2IDsBySubRegionID[ck.subregion] = append(d.dAlpha2IDsBySubRegionID[ck.subregion], k)
		d.dAlpha2IDsByIntermediateRegionID[ck.intregion] = append(d.dAlpha2IDsByIntermediateRegionID[ck.intregion], k)
		d.dAlpha2IDsByStatusID[ck.status] = append(d.dAlpha2IDsByStatusID[ck.status], k)
		d.dAlpha2IDsByTLD[ck.tld] = append(d.dAlpha2IDsByTLD[ck.tld], k)
	}

	delete(d.dAlpha2IDByAlpha3ID, 0)
	delete(d.dAlpha2IDByNumericID, 0)
}

func (d *Data) statusByID(id int) (*enumData, error) {
	if id < 0 || id >= len(d.dStatusByID) {
		return nil, errInvalidKey
	}

	return d.dStatusByID[id], nil
}

func (d *Data) statusIDByName(name string) (uint8, error) {
	v, ok := d.dStatusIDByName[strings.ToUpper(name)]
	if !ok {
		return 0, errInvalidKey
	}

	return v, nil
}

func (d *Data) regionByID(id int) (*enumData, error) {
	if id < 0 || id >= len(d.dRegionByID) {
		return nil, errInvalidKey
	}

	return d.dRegionByID[id], nil
}

func (d *Data) regionIDByCode(code string) (uint8, error) {
	v, ok := d.dRegionIDByCode[code]
	if !ok {
		return 0, errInvalidKey
	}

	return v, nil
}

func (d *Data) regionIDByName(name string) (uint8, error) {
	v, ok := d.dRegionIDByName[strings.ToUpper(name)]
	if !ok {
		return 0, errInvalidKey
	}

	return v, nil
}

func (d *Data) subRegionByID(id int) (*enumData, error) {
	if id < 0 || id >= len(d.dSubRegionByID) {
		return nil, errInvalidKey
	}

	return d.dSubRegionByID[id], nil
}

func (d *Data) subRegionIDByCode(code string) (uint8, error) {
	v, ok := d.dSubRegionIDByCode[code]
	if !ok {
		return 0, errInvalidKey
	}

	return v, nil
}

func (d *Data) subRegionIDByName(name string) (uint8, error) {
	v, ok := d.dSubRegionIDByName[strings.ToUpper(name)]
	if !ok {
		return 0, errInvalidKey
	}

	return v, nil
}

func (d *Data) intermediateRegionByID(id int) (*enumData, error) {
	if id < 0 || id >= len(d.dIntermediateRegionByID) {
		return nil, errInvalidKey
	}

	return d.dIntermediateRegionByID[id], nil
}

func (d *Data) intermediateRegionIDByCode(code string) (uint8, error) {
	v, ok := d.dIntermediateRegionIDByCode[code]
	if !ok {
		return 0, errInvalidKey
	}

	return v, nil
}

func (d *Data) intermediateRegionIDByName(name string) (uint8, error) {
	v, ok := d.dIntermediateRegionIDByName[strings.ToUpper(name)]
	if !ok {
		return 0, errInvalidKey
	}

	return v, nil
}

func (d *Data) countryNamesByAlpha2ID(id uint16) (*Names, error) {
	v, ok := d.dCountryNamesByAlpha2ID[id]
	if !ok {
		return nil, errInvalidKey
	}

	return v, nil
}

func (d *Data) countryKeyByAlpha2ID(id uint16) (uint64, error) {
	v, ok := d.dCountryKeyByAlpha2ID[id]
	if !ok {
		return 0, errInvalidKey
	}

	return v, nil
}

func (d *Data) alpha2IDByAlpha3ID(id uint16) (uint16, error) {
	v, ok := d.dAlpha2IDByAlpha3ID[id]
	if !ok {
		return 0, errInvalidKey
	}

	return v, nil
}

func (d *Data) alpha2IDByNumericID(id uint16) (uint16, error) {
	v, ok := d.dAlpha2IDByNumericID[id]
	if !ok {
		return 0, errInvalidKey
	}

	return v, nil
}

func (d *Data) alpha2IDsByRegionID(id uint8) ([]uint16, error) {
	v, ok := d.dAlpha2IDsByRegionID[id]
	if !ok {
		return nil, errInvalidKey
	}

	return v, nil
}

func (d *Data) alpha2IDsBySubRegionID(id uint8) ([]uint16, error) {
	v, ok := d.dAlpha2IDsBySubRegionID[id]
	if !ok {
		return nil, errInvalidKey
	}

	return v, nil
}

func (d *Data) alpha2IDsByIntermediateRegionID(id uint8) ([]uint16, error) {
	v, ok := d.dAlpha2IDsByIntermediateRegionID[id]
	if !ok {
		return nil, errInvalidKey
	}

	return v, nil
}

func (d *Data) alpha2IDsByStatusID(id uint8) ([]uint16, error) {
	v, ok := d.dAlpha2IDsByStatusID[id]
	if !ok {
		return nil, errInvalidKey
	}

	return v, nil
}

func (d *Data) alpha2IDsByTLD(id uint16) ([]uint16, error) {
	v, ok := d.dAlpha2IDsByTLD[id]
	if !ok {
		return nil, errInvalidKey
	}

	return v, nil
}

// defaultData set the default data.
//
// Default data sources (updated at: 2024-07-17):
//   - https://www.iso.org/iso-3166-country-codes.html
//   - https://www.cia.gov/the-world-factbook/references/country-data-codes/
//   - https://unstats.un.org/unsd/methodology/m49/overview/
//   - https://en.wikipedia.org/wiki/ISO_3166
//   - https://en.wikipedia.org/wiki/ISO_3166-1
//   - https://en.wikipedia.org/wiki/ISO_3166-2
//   - https://en.wikipedia.org/wiki/ISO_3166-3
//   - https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2
//   - https://en.wikipedia.org/wiki/ISO_3166-1_alpha-3
//   - https://en.wikipedia.org/wiki/ISO_3166-1_numeric
//
//nolint:funlen,maintidx
func (d *Data) defaultData() {
	d.dRegionByID = []*enumData{
		{"", ""},
		{"002", "Africa"},
		{"009", "Oceania"},
		{"019", "Americas"},
		{"142", "Asia"},
		{"150", "Europe"},
	}

	d.dSubRegionByID = []*enumData{
		{"", ""},
		{"015", "Northern Africa"},
		{"021", "Northern America"},
		{"030", "Eastern Asia"},
		{"034", "Southern Asia"},
		{"035", "South-eastern Asia"},
		{"039", "Southern Europe"},
		{"053", "Australia and New Zealand"},
		{"054", "Melanesia"},
		{"057", "Micronesia"},
		{"061", "Polynesia"},
		{"143", "Central Asia"},
		{"145", "Western Asia"},
		{"151", "Eastern Europe"},
		{"154", "Northern Europe"},
		{"155", "Western Europe"},
		{"202", "Sub-Saharan Africa"},
		{"419", "Latin America and the Caribbean"},
	}

	d.dIntermediateRegionByID = []*enumData{
		{"", ""},
		{"005", "South America"},
		{"011", "Western Africa"},
		{"013", "Central America"},
		{"014", "Eastern Africa"},
		{"017", "Middle Africa"},
		{"018", "Southern Africa"},
		{"029", "Caribbean"},
	}

	// The key is the alpha-2 ID encoded in 10 bits (see encodeAlpha2).
	d.dCountryNamesByAlpha2ID = map[uint16]*Names{
		0x0024: {
			EN: "Andorra",
			FR: "Andorre (l')",
		},
		0x0025: {
			EN: "United Arab Emirates (the)",
			FR: "Émirats arabes unis (les)",
		},
		0x0026: {
			EN: "Afghanistan",
			FR: "Afghanistan (l')",
		},
		0x0027: {
			EN: "Antigua and Barbuda",
			FR: "Antigua-et-Barbuda",
		},
		0x0029: {
			EN: "Anguilla",
			FR: "Anguilla",
		},
		0x002C: {
			EN: "Albania",
			FR: "Albanie (l')",
		},
		0x002D: {
			EN: "Armenia",
			FR: "Arménie (l')",
		},
		0x002F: {
			EN: "Angola",
			FR: "Angola (l')",
		},
		0x0031: {
			EN: "Antarctica",
			FR: "Antarctique (l')",
		},
		0x0032: {
			EN: "Argentina",
			FR: "Argentine (l')",
		},
		0x0033: {
			EN: "American Samoa",
			FR: "Samoa américaines (les)",
		},
		0x0034: {
			EN: "Austria",
			FR: "Autriche (l')",
		},
		0x0035: {
			EN: "Australia",
			FR: "Australie (l')",
		},
		0x0037: {
			EN: "Aruba",
			FR: "Aruba",
		},
		0x0038: {
			EN: "Åland Islands",
			FR: "Åland(les Îles)",
		},
		0x003A: {
			EN: "Azerbaijan",
			FR: "Azerbaïdjan (l')",
		},
		0x0041: {
			EN: "Bosnia and Herzegovina",
			FR: "Bosnie-Herzégovine (la)",
		},
		0x0042: {
			EN: "Barbados",
			FR: "Barbade (la)",
		},
		0x0044: {
			EN: "Bangladesh",
			FR: "Bangladesh (le)",
		},
		0x0045: {
			EN: "Belgium",
			FR: "Belgique (la)",
		},
		0x0046: {
			EN: "Burkina Faso",
			FR: "Burkina Faso (le)",
		},
		0x0047: {
			EN: "Bulgaria",
			FR: "Bulgarie (la)",
		},
		0x0048: {
			EN: "Bahrain",
			FR: "Bahreïn",
		},
		0x0049: {
			EN: "Burundi",
			FR: "Burundi (le)",
		},
		0x004A: {
			EN: "Benin",
			FR: "Bénin (le)",
		},
		0x004C: {
			EN: "Saint Barthélemy",
			FR: "Saint-Barthélemy",
		},
		0x004D: {
			EN: "Bermuda",
			FR: "Bermudes (les)",
		},
		0x004E: {
			EN: "Brunei Darussalam",
			FR: "Brunéi Darussalam (le)",
		},
		0x004F: {
			EN: "Bolivia (Plurinational State of)",
			FR: "Bolivie (État plurinational de)",
		},
		0x0051: {
			EN: "Bonaire, Sint Eustatius and Saba",
			FR: "Bonaire, Saint-Eustache et Saba",
		},
		0x0052: {
			EN: "Brazil",
			FR: "Brésil (le)",
		},
		0x0053: {
			EN: "Bahamas (the)",
			FR: "Bahamas (les)",
		},
		0x0054: {
			EN: "Bhutan",
			FR: "Bhoutan (le)",
		},
		0x0056: {
			EN: "Bouvet Island",
			FR: "Bouvet (l'Île)",
		},
		0x0057: {
			EN: "Botswana",
			FR: "Botswana (le)",
		},
		0x0059: {
			EN: "Belarus",
			FR: "Bélarus (le)",
		},
		0x005A: {
			EN: "Belize",
			FR: "Belize (le)",
		},
		0x0061: {
			EN: "Canada",
			FR: "Canada (le)",
		},
		0x0063: {
			EN: "Cocos (Keeling) Islands (the)",
			FR: "Cocos (les Îles)/ Keeling (les Îles)",
		},
		0x0064: {
			EN: "Congo (the Democratic Republic of the)",
			FR: "Congo (la République démocratique du)",
		},
		0x0066: {
			EN: "Central African Republic (the)",
			FR: "République centrafricaine (la)",
		},
		0x0067: {
			EN: "Congo (the)",
			FR: "Congo (le)",
		},
		0x0068: {
			EN: "Switzerland",
			FR: "Suisse (la)",
		},
		0x0069: {
			EN: "Côte d'Ivoire",
			FR: "Côte d'Ivoire (la)",
		},
		0x006B: {
			EN: "Cook Islands (the)",
			FR: "Cook (les Îles)",
		},
		0x006C: {
			EN: "Chile",
			FR: "Chili (le)",
		},
		0x006D: {
			EN: "Cameroon",
			FR: "Cameroun (le)",
		},
		0x006E: {
			EN: "China",
			FR: "Chine (la)",
		},
		0x006F: {
			EN: "Colombia",
			FR: "Colombie (la)",
		},
		0x0072: {
			EN: "Costa Rica",
			FR: "Costa Rica (le)",
		},
		0x0075: {
			EN: "Cuba",
			FR: "Cuba",
		},
		0x0076: {
			EN: "Cabo Verde",
			FR: "Cabo Verde",
		},
		0x0077: {
			EN: "Curaçao",
			FR: "Curaçao",
		},
		0x0078: {
			EN: "Christmas Island",
			FR: "Christmas (l'Île)",
		},
		0x0079: {
			EN: "Cyprus",
			FR: "Chypre",
		},
		0x007A: {
			EN: "Czechia",
			FR: "Tchéquie (la)",
		},
		0x0085: {
			EN: "Germany",
			FR: "Allemagne (l')",
		},
		0x008A: {
			EN: "Djibouti",
			FR: "Djibouti",
		},
		0x008B: {
			EN: "Denmark",
			FR: "Danemark (le)",
		},
		0x008D: {
			EN: "Dominica",
			FR: "Dominique (la)",
		},
		0x008F: {
			EN: "Dominican Republic (the)",
			FR: "dominicaine (la République)",
		},
		0x009A: {
			EN: "Algeria",
			FR: "Algérie (l')",
		},
		0x00A3: {
			EN: "Ecuador",
			FR: "Équateur (l')",
		},
		0x00A5: {
			EN: "Estonia",
			FR: "Estonie (l')",
		},
		0x00A7: {
			EN: "Egypt",
			FR: "Égypte (l')",
		},
		0x00A8: {
			EN: "Western Sahara*",
			FR: "Sahara occidental (le)*",
		},
		0x00B2: {
			EN: "Eritrea",
			FR: "Érythrée (l')",
		},
		0x00B3: {
			EN: "Spain",
			FR: "Espagne (l')",
		},
		0x00B4: {
			EN: "Ethiopia",
			FR: "Éthiopie (l')",
		},
		0x00C9: {
			EN: "Finland",
			FR: "Finlande (la)",
		},
		0x00CA: {
			EN: "Fiji",
			FR: "Fidji (les)",
		},
		0x00CB: {
			EN: "Falkland Islands (the) [Malvinas]",
			FR: "Falkland (les Îles)/Malouines (les Îles)",
		},
		0x00CD: {
			EN: "Micronesia (Federated States of)",
			FR: "Micronésie (États fédérés de)",
		},
		0x00CF: {
			EN: "Faroe Islands (the)",
			FR: "Féroé (les Îles)",
		},
		0x00D2: {
			EN: "France",
			FR: "France (la)",
		},
		0x00E1: {
			EN: "Gabon",
			FR: "Gabon (le)",
		},
		0x00E2: {
			EN: "United Kingdom of Great Britain and Northern Ireland (the)",
			FR: "Royaume-Uni de Grande-Bretagne et d'Irlande du Nord (le)",
		},
		0x00E4: {
			EN: "Grenada",
			FR: "Grenade (la)",
		},
		0x00E5: {
			EN: "Georgia",
			FR: "Géorgie (la)",
		},
		0x00E6: {
			EN: "French Guiana",
			FR: "Guyane française (la )",
		},
		0x00E7: {
			EN: "Guernsey",
			FR: "Guernesey",
		},
		0x00E8: {
			EN: "Ghana",
			FR: "Ghana (le)",
		},
		0x00E9: {
			EN: "Gibraltar",
			FR: "Gibraltar",
		},
		0x00EC: {
			EN: "Greenland",
			FR: "Groenland (le)",
		},
		0x00ED: {
			EN: "Gambia (the)",
			FR: "Gambie (la)",
		},
		0x00EE: {
			EN: "Guinea",
			FR: "Guinée (la)",
		},
		0x00F0: {
			EN: "Guadeloupe",
			FR: "Guadeloupe (la)",
		},
		0x00F1: {
			EN: "Equatorial Guinea",
			FR: "Guinée équatoriale (la)",
		},
		0x00F2: {
			EN: "Greece",
			FR: "Grèce (la)",
		},
		0x00F3: {
			EN: "South Georgia and the South Sandwich Islands",
			FR: "Géorgie du Sud-et-les Îles Sandwich du Sud (la)",
		},
		0x00F4: {
			EN: "Guatemala",
			FR: "Guatemala (le)",
		},
		0x00F5: {
			EN: "Guam",
			FR: "Guam",
		},
		0x00F7: {
			EN: "Guinea-Bissau",
			FR: "Guinée-Bissau (la)",
		},
		0x00F9: {
			EN: "Guyana",
			FR: "Guyana (le)",
		},
		0x010B: {
			EN: "Hong Kong",
			FR: "Hong Kong",
		},
		0x010D: {
			EN: "Heard Island and McDonald Islands",
			FR: "Heard-et-Îles MacDonald (l'Île)",
		},
		0x010E: {
			EN: "Honduras",
			FR: "Honduras (le)",
		},
		0x0112: {
			EN: "Croatia",
			FR: "Croatie (la)",
		},
		0x0114: {
			EN: "Haiti",
			FR: "Haïti",
		},
		0x0115: {
			EN: "Hungary",
			FR: "Hongrie (la)",
		},
		0x0124: {
			EN: "Indonesia",
			FR: "Indonésie (l')",
		},
		0x0125: {
			EN: "Ireland",
			FR: "Irlande (l')",
		},
		0x012C: {
			EN: "Israel",
			FR: "Israël",
		},
		0x012D: {
			EN: "Isle of Man",
			FR: "Île de Man",
		},
		0x012E: {
			EN: "India",
			FR: "Inde (l')",
		},
		0x012F: {
			EN: "British Indian Ocean Territory (the)",
			FR: "Indien (le Territoire britannique de l'océan)",
		},
		0x0131: {
			EN: "Iraq",
			FR: "Iraq (l')",
		},
		0x0132: {
			EN: "Iran (Islamic Republic of)",
			FR: "Iran (République Islamique d')",
		},
		0x0133: {
			EN: "Iceland",
			FR: "Islande (l')",
		},
		0x0134: {
			EN: "Italy",
			FR: "Italie (l')",
		},
		0x0145: {
			EN: "Jersey",
			FR: "Jersey",
		},
		0x014D: {
			EN: "Jamaica",
			FR: "Jamaïque (la)",
		},
		0x014F: {
			EN: "Jordan",
			FR: "Jordanie (la)",
		},
		0x0150: {
			EN: "Japan",
			FR: "Japon (le)",
		},
		0x0165: {
			EN: "Kenya",
			FR: "Kenya (le)",
		},
		0x0167: {
			EN: "Kyrgyzstan",
			FR: "Kirghizistan (le)",
		},
		0x0168: {
			EN: "Cambodia",
			FR: "Cambodge (le)",
		},
		0x0169: {
			EN: "Kiribati",
			FR: "Kiribati",
		},
		0x016D: {
			EN: "Comoros (the)",
			FR: "Comores (les)",
		},
		0x016E: {
			EN: "Saint Kitts and Nevis",
			FR: "Saint-Kitts-et-Nevis",
		},
		0x0170: {
			EN: "Korea (the Democratic People's Republic of)",
			FR: "Corée (la République populaire démocratique de)",
		},
		0x0172: {
			EN: "Korea (the Republic of)",
			FR: "Corée (la République de)",
		},
		0x0177: {
			EN: "Kuwait",
			FR: "Koweït (le)",
		},
		0x0179: {
			EN: "Cayman Islands (the)",
			FR: "Caïmans (les Îles)",
		},
		0x017A: {
			EN: "Kazakhstan",
			FR: "Kazakhstan (le)",
		},
		0x0181: {
			EN: "Lao People's Democratic Republic (the)",
			FR: "Lao (la République démocratique populaire)",
		},
		0x0182: {
			EN: "Lebanon",
			FR: "Liban (le)",
		},
		0x0183: {
			EN: "Saint Lucia",
			FR: "Sainte-Lucie",
		},
		0x0189: {
			EN: "Liechtenstein",
			FR: "Liechtenstein (le)",
		},
		0x018B: {
			EN: "Sri Lanka",
			FR: "Sri Lanka",
		},
		0x0192: {
			EN: "Liberia",
			FR: "Libéria (le)",
		},
		0x0193: {
			EN: "Lesotho",
			FR: "Lesotho (le)",
		},
		0x0194: {
			EN: "Lithuania",
			FR: "Lituanie (la)",
		},
		0x0195: {
			EN: "Luxembourg",
			FR: "Luxembourg (le)",
		},
		0x0196: {
			EN: "Latvia",
			FR: "Lettonie (la)",
		},
		0x0199: {
			EN: "Libya",
			FR: "Libye (la)",
		},
		0x01A1: {
			EN: "Morocco",
			FR: "Maroc (le)",
		},
		0x01A3: {
			EN: "Monaco",
			FR: "Monaco",
		},
		0x01A4: {
			EN: "Moldova (the Republic of)",
			FR: "Moldova (la République de)",
		},
		0x01A5: {
			EN: "Montenegro",
			FR: "Monténégro (le)",
		},
		0x01A6: {
			EN: "Saint Martin (French part)",
			FR: "Saint-Martin (partie française)",
		},
		0x01A7: {
			EN: "Madagascar",
			FR: "Madagascar",
		},
		0x01A8: {
			EN: "Marshall Islands (the)",
			FR: "Marshall (les Îles)",
		},
		0x01AB: {
			EN: "North Macedonia",
			FR: "Macédoine du Nord (la)",
		},
		0x01AC: {
			EN: "Mali",
			FR: "Mali (le)",
		},
		0x01AD: {
			EN: "Myanmar",
			FR: "Myanmar (le)",
		},
		0x01AE: {
			EN: "Mongolia",
			FR: "Mongolie (la)",
		},
		0x01AF: {
			EN: "Macao",
			FR: "Macao",
		},
		0x01B0: {
			EN: "Northern Mariana Islands (the)",
			FR: "Mariannes du Nord (les Îles)",
		},
		0x01B1: {
			EN: "Martinique",
			FR: "Martinique (la)",
		},
		0x01B2: {
			EN: "Mauritania",
			FR: "Mauritanie (la)",
		},
		0x01B3: {
			EN: "Montserrat",
			FR: "Montserrat",
		},
		0x01B4: {
			EN: "Malta",
			FR: "Malte",
		},
		0x01B5: {
			EN: "Mauritius",
			FR: "Maurice",
		},
		0x01B6: {
			EN: "Maldives",
			FR: "Maldives (les)",
		},
		0x01B7: {
			EN: "Malawi",
			FR: "Malawi (le)",
		},
		0x01B8: {
			EN: "Mexico",
			FR: "Mexique (le)",
		},
		0x01B9: {
			EN: "Malaysia",
			FR: "Malaisie (la)",
		},
		0x01BA: {
			EN: "Mozambique",
			FR: "Mozambique (le)",
		},
		0x01C1: {
			EN: "Namibia",
			FR: "Namibie (la)",
		},
		0x01C3: {
			EN: "New Caledonia",
			FR: "Nouvelle-Calédonie (la)",
		},
		0x01C5: {
			EN: "Niger (the)",
			FR: "Niger (le)",
		},
		0x01C6: {
			EN: "Norfolk Island",
			FR: "Norfolk (l'Île)",
		},
		0x01C7: {
			EN: "Nigeria",
			FR: "Nigéria (le)",
		},
		0x01C9: {
			EN: "Nicaragua",
			FR: "Nicaragua (le)",
		},
		0x01CC: {
			EN: "Netherlands (Kingdom of the)",
			FR: "Pays-Bas (Royaume des)",
		},
		0x01CF: {
			EN: "Norway",
			FR: "Norvège (la)",
		},
		0x01D0: {
			EN: "Nepal",
			FR: "Népal (le)",
		},
		0x01D2: {
			EN: "Nauru",
			FR: "Nauru",
		},
		0x01D5: {
			EN: "Niue",
			FR: "Niue",
		},
		0x01DA: {
			EN: "New Zealand",
			FR: "Nouvelle-Zélande (la)",
		},
		0x01ED: {
			EN: "Oman",
			FR: "Oman",
		},
		0x0201: {
			EN: "Panama",
			FR: "Panama (le)",
		},
		0x0205: {
			EN: "Peru",
			FR: "Pérou (le)",
		},
		0x0206: {
			EN: "French Polynesia",
			FR: "Polynésie française (la)",
		},
		0x0207: {
			EN: "Papua New Guinea",
			FR: "Papouasie-Nouvelle-Guinée (la)",
		},
		0x0208: {
			EN: "Philippines (the)",
			FR: "Philippines (les)",
		},
		0x020B: {
			EN: "Pakistan",
			FR: "Pakistan (le)",
		},
		0x020C: {
			EN: "Poland",
			FR: "Pologne (la)",
		},
		0x020D: {
			EN: "Saint Pierre and Miquelon",
			FR: "Saint-Pierre-et-Miquelon",
		},
		0x020E: {
			EN: "Pitcairn",
			FR: "Pitcairn",
		},
		0x0212: {
			EN: "Puerto Rico",
			FR: "Porto Rico",
		},
		0x0213: {
			EN: "Palestine, State of",
			FR: "Palestine, État de",
		},
		0x0214: {
			EN: "Portugal",
			FR: "Portugal (le)",
		},
		0x0217: {
			EN: "Palau",
			FR: "Palaos (les)",
		},
		0x0219: {
			EN: "Paraguay",
			FR: "Paraguay (le)",
		},
		0x0221: {
			EN: "Qatar",
			FR: "Qatar (le)",
		},
		0x0245: {
			EN: "Réunion",
			FR: "Réunion (La)",
		},
		0x024F: {
			EN: "Romania",
			FR: "Roumanie (la)",
		},
		0x0253: {
			EN: "Serbia",
			FR: "Serbie (la)",
		},
		0x0255: {
			EN: "Russian Federation (the)",
			FR: "Russie (la Fédération de)",
		},
		0x0257: {
			EN: "Rwanda",
			FR: "Rwanda (le)",
		},
		0x0261: {
			EN: "Saudi Arabia",
			FR: "Arabie saoudite (l')",
		},
		0x0262: {
			EN: "Solomon Islands",
			FR: "Salomon (les Îles)",
		},
		0x0263: {
			EN: "Seychelles",
			FR: "Seychelles (les)",
		},
		0x0264: {
			EN: "Sudan (the)",
			FR: "Soudan (le)",
		},
		0x0265: {
			EN: "Sweden",
			FR: "Suède (la)",
		},
		0x0267: {
			EN: "Singapore",
			FR: "Singapour",
		},
		0x0268: {
			EN: "Saint Helena, Ascension and Tristan da Cunha",
			FR: "Sainte-Hélène, Ascension et Tristan da Cunha",
		},
		0x0269: {
			EN: "Slovenia",
			FR: "Slovénie (la)",
		},
		0x026A: {
			EN: "Svalbard and Jan Mayen",
			FR: "Svalbard et l'Île Jan Mayen (le)",
		},
		0x026B: {
			EN: "Slovakia",
			FR: "Slovaquie (la)",
		},
		0x026C: {
			EN: "Sierra Leone",
			FR: "Sierra Leone (la)",
		},
		0x026D: {
			EN: "San Marino",
			FR: "Saint-Marin",
		},
		0x026E: {
			EN: "Senegal",
			FR: "Sénégal (le)",
		},
		0x026F: {
			EN: "Somalia",
			FR: "Somalie (la)",
		},
		0x0272: {
			EN: "Suriname",
			FR: "Suriname (le)",
		},
		0x0273: {
			EN: "South Sudan",
			FR: "Soudan du Sud (le)",
		},
		0x0274: {
			EN: "Sao Tome and Principe",
			FR: "Sao Tomé-et-Principe",
		},
		0x0276: {
			EN: "El Salvador",
			FR: "El Salvador",
		},
		0x0278: {
			EN: "Sint Maarten (Dutch part)",
			FR: "Saint-Martin (partie néerlandaise)",
		},
		0x0279: {
			EN: "Syrian Arab Republic (the)",
			FR: "République arabe syrienne (la)",
		},
		0x027A: {
			EN: "Eswatini",
			FR: "Eswatini (l')",
		},
		0x0283: {
			EN: "Turks and Caicos Islands (the)",
			FR: "Turks-et-Caïcos (les Îles)",
		},
		0x0284: {
			EN: "Chad",
			FR: "Tchad (le)",
		},
		0x0286: {
			EN: "French Southern Territories (the)",
			FR: "Terres australes françaises (les)",
		},
		0x0287: {
			EN: "Togo",
			FR: "Togo (le)",
		},
		0x0288: {
			EN: "Thailand",
			FR: "Thaïlande (la)",
		},
		0x028A: {
			EN: "Tajikistan",
			FR: "Tadjikistan (le)",
		},
		0x028B: {
			EN: "Tokelau",
			FR: "Tokelau (les)",
		},
		0x028C: {
			EN: "Timor-Leste",
			FR: "Timor-Leste (le)",
		},
		0x028D: {
			EN: "Turkmenistan",
			FR: "Turkménistan (le)",
		},
		0x028E: {
			EN: "Tunisia",
			FR: "Tunisie (la)",
		},
		0x028F: {
			EN: "Tonga",
			FR: "Tonga (les)",
		},
		0x0292: {
			EN: "Türkiye",
			FR: "Türkiye (la)",
		},
		0x0294: {
			EN: "Trinidad and Tobago",
			FR: "Trinité-et-Tobago (la)",
		},
		0x0296: {
			EN: "Tuvalu",
			FR: "Tuvalu (les)",
		},
		0x0297: {
			EN: "Taiwan (Province of China)",
			FR: "Taïwan (Province de Chine)",
		},
		0x029A: {
			EN: "Tanzania, the United Republic of",
			FR: "Tanzanie (la République-Unie de)",
		},
		0x02A1: {
			EN: "Ukraine",
			FR: "Ukraine (l')",
		},
		0x02A7: {
			EN: "Uganda",
			FR: "Ouganda (l')",
		},
		0x02AD: {
			EN: "United States Minor Outlying Islands (the)",
			FR: "Îles mineures éloignées des États-Unis (les)",
		},
		0x02B3: {
			EN: "United States of America (the)",
			FR: "États-Unis d'Amérique (les)",
		},
		0x02B9: {
			EN: "Uruguay",
			FR: "Uruguay (l')",
		},
		0x02BA: {
			EN: "Uzbekistan",
			FR: "Ouzbékistan (l')",
		},
		0x02C1: {
			EN: "Holy See (the)",
			FR: "Saint-Siège (le)",
		},
		0x02C3: {
			EN: "Saint Vincent and the Grenadines",
			FR: "Saint-Vincent-et-les Grenadines",
		},
		0x02C5: {
			EN: "Venezuela (Bolivarian Republic of)",
			FR: "Venezuela (République bolivarienne du)",
		},
		0x02C7: {
			EN: "Virgin Islands (British)",
			FR: "Vierges britanniques (les Îles)",
		},
		0x02C9: {
			EN: "Virgin Islands (U.S.)",
			FR: "Vierges des États-Unis (les Îles)",
		},
		0x02CE: {
			EN: "Viet Nam",
			FR: "Viet Nam (le)",
		},
		0x02D5: {
			EN: "Vanuatu",
			FR: "Vanuatu (le)",
		},
		0x02E6: {
			EN: "Wallis and Futuna",
			FR: "Wallis-et-Futuna ",
		},
		0x02F3: {
			EN: "Samoa",
			FR: "Samoa (le)",
		},
		0x0325: {
			EN: "Yemen",
			FR: "Yémen (le)",
		},
		0x0334: {
			EN: "Mayotte",
			FR: "Mayotte",
		},
		0x0341: {
			EN: "South Africa",
			FR: "Afrique du Sud (l')",
		},
		0x034D: {
			EN: "Zambia",
			FR: "Zambie (la)",
		},
		0x0357: {
			EN: "Zimbabwe",
			FR: "Zimbabwe (le)",
		},
	}

	// dCountryKeyByAlpha2ID contains all the countries data, excluding names, in a compact form.
	// The key is the alpha-2 code encoded in 10 bits (see encodeAlpha2).
	// The value is the country data encoded in 64 bits (see decodeCountryKey).
	// This map contains all the 2-characters combinations from AA to ZZ.
	d.dCountryKeyByAlpha2ID = map[uint16]uint64{
		0x0021: 0x2084000000000000,
		0x0022: 0x0088000000000000,
		0x0023: 0x308C000000000000,
		0x0024: 0x10902E2028530024,
		0x0025: 0x1094322E20460025,
		0x0026: 0x1098263808420026,
		0x0027: 0x109C343838389C27,
		0x0028: 0x00A0000000000000,
		0x0029: 0x10A4290D28389C29,
		0x002A: 0x00A8000000000000,
		0x002B: 0x00AC000000000000,
		0x002C: 0x10B02C101053002C,
		0x002D: 0x10B432686646002D,
		0x002E: 0x40B8000000000000,
		0x002F: 0x10BC27783018142F,
		0x0030: 0x50C0000000000000,
		0x0031: 0x10C4340814000031,
		0x0032: 0x10C8323840388432,
		0x0033: 0x10CC336820250033,
		0x0034: 0x10D035A050578034,
		0x0035: 0x10D4359848238035,
		0x0036: 0x00D8000000000000,
		0x0037: 0x10DC22BC2A389C37,
		0x0038: 0x10E02C09F0570038,
		0x0039: 0x00E4000000000000,
		0x003A: 0x10E83A283E46003A,
		0x0041: 0x110449408C530041,
		0x0042: 0x1108521068389C42,
		0x0043: 0x010C000000000000,
		0x0044: 0x1110472064420044,
		0x0045: 0x1114456070578045,
		0x0046: 0x1118460EAC180846,
		0x0047: 0x111C4790C8568047,
		0x0048: 0x1120489060460048,
		0x0049: 0x11244448D8181049,
		0x004A: 0x112845719818084A,
		0x004B: 0x012C000000000000,
		0x004C: 0x11304C6D18389C4C,
		0x004D: 0x11344DA87831004D,
		0x004E: 0x11385270C042804E,
		0x004F: 0x113C4F608838844F,
		0x0050: 0x0140000000000000,
		0x0051: 0x1144459C2E389C51,
		0x0052: 0x1148520898388452,
		0x0053: 0x114C489858389C53,
		0x0054: 0x1150547080420054,
		0x0055: 0x4154000000000000,
		0x0056: 0x115856A094388456,
		0x0057: 0x115C570890181857,
		0x0058: 0x5160000000000000,
		0x0059: 0x11644C90E0568059,
		0x005A: 0x11684CD0A8388C5A,
		0x0061: 0x11846170F8310061,
		0x0062: 0x0188000000000000,
		0x0063: 0x118C63594C238063,
		0x0064: 0x11906F2168181464,
		0x0065: 0x0194000000000000,
		0x0066: 0x1198613118181466,
		0x0067: 0x119C6F3964181467,
		0x0068: 0x11A0682DE8578068,
		0x0069: 0x11A469B300180869,
		0x006A: 0x01A8000000000000,
		0x006B: 0x11AC6F597025006B,
		0x006C: 0x11B068613038846C,
		0x006D: 0x11B46D90F018146D,
		0x006E: 0x11B868713841806E,
		0x006F: 0x11BC6F615438846F,
		0x0070: 0x31C0000000000000,
		0x0071: 0x31C4000000000000,
		0x0072: 0x11C8724978388C72,
		0x0073: 0x41CC000000000000,
		0x0074: 0x61D0000000000000,
		0x0075: 0x11D4751180389C75,
		0x0076: 0x11D870B108180876,
		0x0077: 0x11DC75BC26389C77,
		0x0078: 0x11E0789144238078,
		0x0079: 0x11E4798188460079,
		0x007A: 0x11E87A299656807A,
		0x0081: 0x0204000000000000,
		0x0082: 0x0208000000000000,
		0x0083: 0x020C000000000000,
		0x0084: 0x6210000000000000,
		0x0085: 0x121485AA28578085,
		0x0086: 0x0218000000000000,
		0x0087: 0x321C000000000000,
		0x0088: 0x0220000000000000,
		0x0089: 0x0224000000000000,
		0x008A: 0x12288A4A0C18108A,
		0x008B: 0x122C8E59A057008B,
		0x008C: 0x0230000000000000,
		0x008D: 0x12348D09A8389C8D,
		0x008E: 0x0238000000000000,
		0x008F: 0x123C8F69AC389C8F,
		0x0090: 0x0240000000000000,
		0x0091: 0x0244000000000000,
		0x0092: 0x0248000000000000,
		0x0093: 0x024C000000000000,
		0x0094: 0x0250000000000000,
		0x0095: 0x0254000000000000,
		0x0096: 0x0258000000000000,
		0x0097: 0x025C000000000000,
		0x0098: 0x0260000000000000,
		0x0099: 0x5264000000000000,
		0x009A: 0x12689A081810809A,
		0x00A1: 0x3284000000000000,
		0x00A2: 0x0288000000000000,
		0x00A3: 0x128CA3A9B43884A3,
		0x00A4: 0x0290000000000000,
		0x00A5: 0x1294B3A1D25700A5,
		0x00A6: 0x5298000000000000,
		0x00A7: 0x129CA7CE641080A7,
		0x00A8: 0x12A0B345B81080A8,
		0x00A9: 0x02A4000000000000,
		0x00AA: 0x02A8000000000000,
		0x00AB: 0x02AC000000000000,
		0x00AC: 0x02B0000000000000,
		0x00AD: 0x52B4000000000000,
		0x00AE: 0x02B8000000000000,
		0x00AF: 0x02BC000000000000,
		0x00B0: 0x52C0000000000000,
		0x00B1: 0x02C4000000000000,
		0x00B2: 0x12C8B249D01810B2,
		0x00B3: 0x12CCB385A85300B3,
		0x00B4: 0x12D0B441CE1810B4,
		0x00B5: 0x32D4000000000000,
		0x00B6: 0x52D8000000000000,
		0x00B7: 0x52DC000000000000,
		0x00B8: 0x02E0000000000000,
		0x00B9: 0x02E4000000000000,
		0x00BA: 0x32E8000000000000,
		0x00C1: 0x0304000000000000,
		0x00C2: 0x0308000000000000,
		0x00C3: 0x030C000000000000,
		0x00C4: 0x0310000000000000,
		0x00C5: 0x0314000000000000,
		0x00C6: 0x0318000000000000,
		0x00C7: 0x031C000000000000,
		0x00C8: 0x0320000000000000,
		0x00C9: 0x1324C971EC5700C9,
		0x00CA: 0x1328CA49E42400CA,
		0x00CB: 0x132CCC59DC3884CB,
		0x00CC: 0x5330000000000000,
		0x00CD: 0x1334D36C8E2480CD,
		0x00CE: 0x0338000000000000,
		0x00CF: 0x133CD279D45700CF,
		0x00D0: 0x0340000000000000,
		0x00D1: 0x6344000000000000,
		0x00D2: 0x1348D209F45780D2,
		0x00D3: 0x034C000000000000,
		0x00D4: 0x0350000000000000,
		0x00D5: 0x0354000000000000,
		0x00D6: 0x0358000000000000,
		0x00D7: 0x035C000000000000,
		0x00D8: 0x3360000000000000,
		0x00D9: 0x0364000000000000,
		0x00DA: 0x0368000000000000,
		0x00E1: 0x1384E112141814E1,
		0x00E2: 0x1388E296745702AB,
		0x00E3: 0x538C000000000000,
		0x00E4: 0x1390F22268389CE4,
		0x00E5: 0x1394E57A184600E5,
		0x00E6: 0x1398F531FC3884E6,
		0x00E7: 0x139CE7CE7E5700E7,
		0x00E8: 0x13A0E80A401808E8,
		0x00E9: 0x13A4E912485300E9,
		0x00EA: 0x03A8000000000000,
		0x00EB: 0x03AC000000000000,
		0x00EC: 0x13B0F262603100EC,
		0x00ED: 0x13B4ED121C1808ED,
		0x00EE: 0x13B8E972881808EE,
		0x00EF: 0x03BC000000000000,
		0x00F0: 0x13C0EC8270389CF0,
		0x00F1: 0x13C4EE89C41814F1,
		0x00F2: 0x13C8F21A585300F2,
		0x00F3: 0x13CE6799DE3884F3,
		0x00F4: 0x13D0F46A80388CF4,
		0x00F5: 0x13D4F56A782480F5,
		0x00F6: 0x03D8000000000000,
		0x00F7: 0x13DCEE14E01808F7,
		0x00F8: 0x03E0000000000000,
		0x00F9: 0x13E4F5CA903884F9,
		0x00FA: 0x03E8000000000000,
		0x0101: 0x0404000000000000,
		0x0102: 0x0408000000000000,
		0x0103: 0x040C000000000000,
		0x0104: 0x0410000000000000,
		0x0105: 0x0414000000000000,
		0x0106: 0x0418000000000000,
		0x0107: 0x041C000000000000,
		0x0108: 0x0420000000000000,
		0x0109: 0x0424000000000000,
		0x010A: 0x0428000000000000,
		0x010B: 0x142D0B3AB041810B,
		0x010C: 0x0430000000000000,
		0x010D: 0x14350D229C23810D,
		0x010E: 0x14390E22A8388D0E,
		0x010F: 0x043C000000000000,
		0x0110: 0x0440000000000000,
		0x0111: 0x0444000000000000,
		0x0112: 0x144912B17E530112,
		0x0113: 0x044C000000000000,
		0x0114: 0x1451144A98389D14,
		0x0115: 0x14551572B8568115,
		0x0116: 0x6458000000000000,
		0x0117: 0x045C000000000000,
		0x0118: 0x0460000000000000,
		0x0119: 0x0464000000000000,
		0x011A: 0x0468000000000000,
		0x0121: 0x0484000000000000,
		0x0122: 0x5488000000000000,
		0x0123: 0x348C000000000000,
		0x0124: 0x14912472D0428124,
		0x0125: 0x14953262E8570125,
		0x0126: 0x0498000000000000,
		0x0127: 0x049C000000000000,
		0x0128: 0x04A0000000000000,
		0x0129: 0x04A4000000000000,
		0x012A: 0x04A8000000000000,
		0x012B: 0x04AC000000000000,
		0x012C: 0x14B13392F046012C,
		0x012D: 0x14B52D768257012D,
		0x012E: 0x14B92E22C842012E,
		0x012F: 0x14BD2FA0AC18112F,
		0x0130: 0x04C0000000000000,
		0x0131: 0x14C5328AE0460131,
		0x0132: 0x14C93272D8420132,
		0x0133: 0x14CD3362C0570133,
		0x0134: 0x14D1340AF8530134,
		0x0135: 0x04D4000000000000,
		0x0136: 0x04D8000000000000,
		0x0137: 0x04DC000000000000,
		0x0138: 0x04E0000000000000,
		0x0139: 0x04E4000000000000,
		0x013A: 0x04E8000000000000,
		0x0141: 0x5504000000000000,
		0x0142: 0x0508000000000000,
		0x0143: 0x050C000000000000,
		0x0144: 0x0510000000000000,
		0x0145: 0x151545CE80570145,
		0x0146: 0x0518000000000000,
		0x0147: 0x051C000000000000,
		0x0148: 0x0520000000000000,
		0x0149: 0x0524000000000000,
		0x014A: 0x0528000000000000,
		0x014B: 0x052C000000000000,
		0x014C: 0x0530000000000000,
		0x014D: 0x1535416B08389D4D,
		0x014E: 0x0538000000000000,
		0x014F: 0x153D4F932046014F,
		0x0150: 0x1541507310418150,
		0x0151: 0x0544000000000000,
		0x0152: 0x0548000000000000,
		0x0153: 0x054C000000000000,
		0x0154: 0x6550000000000000,
		0x0155: 0x0554000000000000,
		0x0156: 0x0558000000000000,
		0x0157: 0x055C000000000000,
		0x0158: 0x0560000000000000,
		0x0159: 0x0564000000000000,
		0x015A: 0x0568000000000000,
		0x0161: 0x0584000000000000,
		0x0162: 0x0588000000000000,
		0x0163: 0x058C000000000000,
		0x0164: 0x0590000000000000,
		0x0165: 0x1595657328181165,
		0x0166: 0x0598000000000000,
		0x0167: 0x159D67D342458167,
		0x0168: 0x15A16868E8428168,
		0x0169: 0x15A5699250248169,
		0x016A: 0x05A8000000000000,
		0x016B: 0x05AC000000000000,
		0x016C: 0x05B0000000000000,
		0x016D: 0x15B46F695C18116D,
		0x016E: 0x15B96E0D26389D6E,
		0x016F: 0x05BC000000000000,
		0x0170: 0x15C2125B30418170,
		0x0171: 0x05C4000000000000,
		0x0172: 0x15C96F9334418172,
		0x0173: 0x05CC000000000000,
		0x0174: 0x05D0000000000000,
		0x0175: 0x05D4000000000000,
		0x0176: 0x05D8000000000000,
		0x0177: 0x15DD77A33C460177,
		0x0178: 0x05E0000000000000,
		0x0179: 0x15E4796910389D79,
		0x017A: 0x15E961D31C45817A,
		0x0181: 0x1605817B44428181,
		0x0182: 0x160982734C460182,
		0x0183: 0x160D830D2C389D83,
		0x0184: 0x0610000000000000,
		0x0185: 0x0614000000000000,
		0x0186: 0x5618000000000000,
		0x0187: 0x061C000000000000,
		0x0188: 0x0620000000000000,
		0x0189: 0x1625892B6C578189,
		0x018A: 0x0628000000000000,
		0x018B: 0x162D8B092042018B,
		0x018C: 0x0630000000000000,
		0x018D: 0x0634000000000000,
		0x018E: 0x0638000000000000,
		0x018F: 0x063C000000000000,
		0x0190: 0x0640000000000000,
		0x0191: 0x0644000000000000,
		0x0192: 0x164982935C180992,
		0x0193: 0x164D937B54181993,
		0x0194: 0x165194AB70570194,
		0x0195: 0x165595C374578195,
		0x0196: 0x1659960B58570196,
		0x0197: 0x065C000000000000,
		0x0198: 0x0660000000000000,
		0x0199: 0x166582CB64108199,
		0x019A: 0x0668000000000000,
		0x01A1: 0x1685A193F01081A1,
		0x01A2: 0x0688000000000000,
		0x01A3: 0x168DA37BD85781A3,
		0x01A4: 0x1691A40BE45681A4,
		0x01A5: 0x1695AE2BE65301A5,
		0x01A6: 0x1699A1352E389DA6,
		0x01A7: 0x169DA43B841811A7,
		0x01A8: 0x16A1A864902481A8,
		0x01A9: 0x66A4000000000000,
		0x01AA: 0x06A8000000000000,
		0x01AB: 0x16ADAB264E5301AB,
		0x01AC: 0x16B1AC4BA41809AC,
		0x01AD: 0x16B5AD90D04281AD,
		0x01AE: 0x16B9AE3BE04181AE,
		0x01AF: 0x16BDA11B7C4181AF,
		0x01B0: 0x16C1AE84882481B0,
		0x01B1: 0x16C5B48BB4389DB1,
		0x01B2: 0x16C9B2A3BC1809B2,
		0x01B3: 0x16CDB393E8389DB3,
		0x01B4: 0x16D1ACA3AC5301B4,
		0x01B5: 0x16D5B59BC01811B5,
		0x01B6: 0x16D9A4B39C4201B6,
		0x01B7: 0x16DDB74B8C1811B7,
		0x01B8: 0x16E1A5C3C8388DB8,
		0x01B9: 0x16E5B99B944281B9,
		0x01BA: 0x16E9AFD3F81811BA,
		0x01C1: 0x1705C16C081819C1,
		0x01C2: 0x0708000000000000,
		0x01C3: 0x170DC364382401C3,
		0x01C4: 0x0710000000000000,
		0x01C5: 0x1715C594641809C5,
		0x01C6: 0x1719C65C7C2381C6,
		0x01C7: 0x171DC70C6C1809C7,
		0x01C8: 0x6720000000000000,
		0x01C9: 0x1725C91C5C388DC9,
		0x01CA: 0x0728000000000000,
		0x01CB: 0x072C000000000000,
		0x01CC: 0x1731CC24205781CC,
		0x01CD: 0x0734000000000000,
		0x01CE: 0x0738000000000000,
		0x01CF: 0x173DCF94845701CF,
		0x01D0: 0x1741D064184201D0,
		0x01D1: 0x6744000000000000,
		0x01D2: 0x1749D2AC102481D2,
		0x01D3: 0x074C000000000000,
		0x01D4: 0x4750000000000000,
		0x01D5: 0x1755C9AC742501D5,
		0x01D6: 0x0758000000000000,
		0x01D7: 0x075C000000000000,
		0x01D8: 0x0760000000000000,
		0x01D9: 0x0764000000000000,
		0x01DA: 0x1769DA64542381DA,
		0x01E1: 0x5784000000000000,
		0x01E2: 0x0788000000000000,
		0x01E3: 0x078C000000000000,
		0x01E4: 0x0790000000000000,
		0x01E5: 0x0794000000000000,
		0x01E6: 0x0798000000000000,
		0x01E7: 0x079C000000000000,
		0x01E8: 0x07A0000000000000,
		0x01E9: 0x07A4000000000000,
		0x01EA: 0x07A8000000000000,
		0x01EB: 0x07AC000000000000,
		0x01EC: 0x07B0000000000000,
		0x01ED: 0x17B5ED74004601ED,
		0x01EE: 0x07B8000000000000,
		0x01EF: 0x07BC000000000000,
		0x01F0: 0x07C0000000000000,
		0x01F1: 0x07C4000000000000,
		0x01F2: 0x07C8000000000000,
		0x01F3: 0x07CC000000000000,
		0x01F4: 0x07D0000000000000,
		0x01F5: 0x07D4000000000000,
		0x01F6: 0x07D8000000000000,
		0x01F7: 0x07DC000000000000,
		0x01F8: 0x07E0000000000000,
		0x01F9: 0x07E4000000000000,
		0x01FA: 0x07E8000000000000,
		0x0201: 0x180601749E388E01,
		0x0202: 0x0808000000000000,
		0x0203: 0x680C000000000000,
		0x0204: 0x0810000000000000,
		0x0205: 0x18160594B8388605,
		0x0206: 0x181A193204250206,
		0x0207: 0x181E0E3CAC240207,
		0x0208: 0x18220864C0428208,
		0x0209: 0x5824000000000000,
		0x020A: 0x0828000000000000,
		0x020B: 0x182E015C9442020B,
		0x020C: 0x18320F64D056820C,
		0x020D: 0x1836706D3431020D,
		0x020E: 0x183A0374C825020E,
		0x020F: 0x083C000000000000,
		0x0210: 0x0840000000000000,
		0x0211: 0x0844000000000000,
		0x0212: 0x184A124CEC389E12,
		0x0213: 0x184E132A26460213,
		0x0214: 0x185212A4D8530214,
		0x0215: 0x6854000000000000,
		0x0216: 0x0858000000000000,
		0x0217: 0x185E0CBC92248217,
		0x0218: 0x0860000000000000,
		0x0219: 0x186612CCB0388619,
		0x021A: 0x6868000000000000,
		0x0221: 0x188621A4F4460221,
		0x0222: 0x0888000000000000,
		0x0223: 0x088C000000000000,
		0x0224: 0x0890000000000000,
		0x0225: 0x0894000000000000,
		0x0226: 0x0898000000000000,
		0x0227: 0x089C000000000000,
		0x0228: 0x08A0000000000000,
		0x0229: 0x08A4000000000000,
		0x022A: 0x08A8000000000000,
		0x022B: 0x08AC000000000000,
		0x022C: 0x08B0000000000000,
		0x022D: 0x28B4000000000000,
		0x022E: 0x28B8000000000000,
		0x022F: 0x28BC000000000000,
		0x0230: 0x28C0000000000000,
		0x0231: 0x28C4000000000000,
		0x0232: 0x28C8000000000000,
		0x0233: 0x28CC000000000000,
		0x0234: 0x28D0000000000000,
		0x0235: 0x28D4000000000000,
		0x0236: 0x28D8000000000000,
		0x0237: 0x28DC000000000000,
		0x0238: 0x28E0000000000000,
		0x0239: 0x28E4000000000000,
		0x023A: 0x28E8000000000000,
		0x0241: 0x5904000000000000,
		0x0242: 0x5908000000000000,
		0x0243: 0x590C000000000000,
		0x0244: 0x0910000000000000,
		0x0245: 0x191645ACFC181245,
		0x0246: 0x0918000000000000,
		0x0247: 0x091C000000000000,
		0x0248: 0x5920000000000000,
		0x0249: 0x5924000000000000,
		0x024A: 0x0928000000000000,
		0x024B: 0x092C000000000000,
		0x024C: 0x5930000000000000,
		0x024D: 0x5934000000000000,
		0x024E: 0x5938000000000000,
		0x024F: 0x193E4FAD0456824F,
		0x0250: 0x5940000000000000,
		0x0251: 0x0944000000000000,
		0x0252: 0x0948000000000000,
		0x0253: 0x194E721560530253,
		0x0254: 0x0950000000000000,
		0x0255: 0x1956559D06568255,
		0x0256: 0x0958000000000000,
		0x0257: 0x195E570D0C181257,
		0x0258: 0x0960000000000000,
		0x0259: 0x0964000000000000,
		0x025A: 0x0968000000000000,
		0x0261: 0x198661AD54460261,
		0x0262: 0x198A6C10B4240262,
		0x0263: 0x198E791D64181263,
		0x0264: 0x19926475B2108264,
		0x0265: 0x1996772DE0570265,
		0x0266: 0x5998000000000000,
		0x0267: 0x199E67857C428267,
		0x0268: 0x19A268751C180A68,
		0x0269: 0x19A6767582530269,
		0x026A: 0x19AA6A6DD057026A,
		0x026B: 0x19AE765D7E56826B,
		0x026C: 0x19B26C2D6C180A6C,
		0x026D: 0x19B66D954453026D,
		0x026E: 0x19BA65755C180A6E,
		0x026F: 0x19BE6F6D8418126F,
		0x0270: 0x09C0000000000000,
		0x0271: 0x09C4000000000000,
		0x0272: 0x19CA7595C8388672,
		0x0273: 0x19CE7325B0181273,
		0x0274: 0x19D274854C181674,
		0x0275: 0x39D4000000000000,
		0x0276: 0x19DA6CB1BC388E76,
		0x0277: 0x09DC000000000000,
		0x0278: 0x19E2786C2C389E78,
		0x0279: 0x19E67995F0460279,
		0x027A: 0x19EA77D5D8181A7A,
		0x0281: 0x3A04000000000000,
		0x0282: 0x0A08000000000000,
		0x0283: 0x1A0E830E38389E83,
		0x0284: 0x1A12832128181684,
		0x0285: 0x0A14000000000000,
		0x0286: 0x1A18343208181286,
		0x0287: 0x1A1E877E00180A87,
		0x0288: 0x1A22880DF8428288,
		0x0289: 0x0A24000000000000,
		0x028A: 0x1A2A8A5DF445828A,
		0x028B: 0x1A2E8B660825028B,
		0x028C: 0x1A328C9CE442828C,
		0x028D: 0x1A368B6E3645828D,
		0x028E: 0x1A3A95762810828E,
		0x028F: 0x1A3E8F761025028F,
		0x0290: 0x4A40000000000000,
		0x0291: 0x0A44000000000000,
		0x0292: 0x1A4A959630460292,
		0x0293: 0x0A4C000000000000,
		0x0294: 0x1A52947E18389E94,
		0x0295: 0x0A54000000000000,
		0x0296: 0x1A5A95B63C250296,
		0x0297: 0x1A5E97713C000297,
		0x0298: 0x0A60000000000000,
		0x0299: 0x0A64000000000000,
		0x029A: 0x1A6A9A0E8418129A,
		0x02A1: 0x1A86AB96485682A1,
		0x02A2: 0x0A88000000000000,
		0x02A3: 0x0A8C000000000000,
		0x02A4: 0x0A90000000000000,
		0x02A5: 0x0A94000000000000,
		0x02A6: 0x0A98000000000000,
		0x02A7: 0x1A9EA70E401812A7,
		0x02A8: 0x0AA0000000000000,
		0x02A9: 0x0AA4000000000000,
		0x02AA: 0x0AA8000000000000,
		0x02AB: 0x3AAC000000000000,
		0x02AC: 0x0AB0000000000000,
		0x02AD: 0x1AB6AD4C8A2482B3,
		0x02AE: 0x3AB8000000000000,
		0x02AF: 0x0ABC000000000000,
		0x02B0: 0x0AC0000000000000,
		0x02B1: 0x0AC4000000000000,
		0x02B2: 0x0AC8000000000000,
		0x02B3: 0x1ACEB30E903102B3,
		0x02B4: 0x0AD0000000000000,
		0x02B5: 0x0AD4000000000000,
		0x02B6: 0x0AD8000000000000,
		0x02B7: 0x0ADC000000000000,
		0x02B8: 0x0AE0000000000000,
		0x02B9: 0x1AE6B2CEB43886B9,
		0x02BA: 0x1AEABA16B84582BA,
		0x02C1: 0x1B06C1A2A05302C1,
		0x02C2: 0x0B08000000000000,
		0x02C3: 0x1B0EC3A53C389EC3,
		0x02C4: 0x6B10000000000000,
		0x02C5: 0x1B16C576BC3886C5,
		0x02C6: 0x0B18000000000000,
		0x02C7: 0x1B1EC710B8389EC7,
		0x02C8: 0x0B20000000000000,
		0x02C9: 0x1B26C996A4389EC9,
		0x02CA: 0x0B28000000000000,
		0x02CB: 0x0B2C000000000000,
		0x02CC: 0x0B30000000000000,
		0x02CD: 0x0B34000000000000,
		0x02CE: 0x1B3ACE6D804282CE,
		0x02CF: 0x0B3C000000000000,
		0x02D0: 0x0B40000000000000,
		0x02D1: 0x0B44000000000000,
		0x02D2: 0x0B48000000000000,
		0x02D3: 0x0B4C000000000000,
		0x02D4: 0x0B50000000000000,
		0x02D5: 0x1B56D5A4482402D5,
		0x02D6: 0x0B58000000000000,
		0x02D7: 0x0B5C000000000000,
		0x02D8: 0x0B60000000000000,
		0x02D9: 0x0B64000000000000,
		0x02DA: 0x0B68000000000000,
		0x02E1: 0x0B84000000000000,
		0x02E2: 0x0B88000000000000,
		0x02E3: 0x0B8C000000000000,
		0x02E4: 0x0B90000000000000,
		0x02E5: 0x0B94000000000000,
		0x02E6: 0x1B9AEC36D82502E6,
		0x02E7: 0x5B9C000000000000,
		0x02E8: 0x0BA0000000000000,
		0x02E9: 0x0BA4000000000000,
		0x02EA: 0x0BA8000000000000,
		0x02EB: 0x6BAC000000000000,
		0x02EC: 0x5BB0000000000000,
		0x02ED: 0x0BB4000000000000,
		0x02EE: 0x0BB8000000000000,
		0x02EF: 0x5BBC000000000000,
		0x02F0: 0x0BC0000000000000,
		0x02F1: 0x0BC4000000000000,
		0x02F2: 0x0BC8000000000000,
		0x02F3: 0x1BCEF36EE42502F3,
		0x02F4: 0x0BD0000000000000,
		0x02F5: 0x0BD4000000000000,
		0x02F6: 0x5BD8000000000000,
		0x02F7: 0x0BDC000000000000,
		0x02F8: 0x0BE0000000000000,
		0x02F9: 0x0BE4000000000000,
		0x02FA: 0x0BE8000000000000,
		0x0301: 0x2C04000000000000,
		0x0302: 0x2C08000000000000,
		0x0303: 0x2C0C000000000000,
		0x0304: 0x2C10000000000000,
		0x0305: 0x2C14000000000000,
		0x0306: 0x2C18000000000000,
		0x0307: 0x2C1C000000000000,
		0x0308: 0x2C20000000000000,
		0x0309: 0x2C24000000000000,
		0x030A: 0x2C28000000000000,
		0x030B: 0x2C2C00000000016F,
		0x030C: 0x2C30000000000000,
		0x030D: 0x2C34000000000000,
		0x030E: 0x2C38000000000000,
		0x030F: 0x2C3C000000000000,
		0x0310: 0x2C40000000000000,
		0x0311: 0x2C44000000000000,
		0x0312: 0x2C48000000000000,
		0x0313: 0x2C4C000000000000,
		0x0314: 0x2C50000000000000,
		0x0315: 0x2C54000000000000,
		0x0316: 0x2C58000000000000,
		0x0317: 0x2C5C000000000000,
		0x0318: 0x2C60000000000000,
		0x0319: 0x2C64000000000000,
		0x031A: 0x2C68000000000000,
		0x0321: 0x0C84000000000000,
		0x0322: 0x0C88000000000000,
		0x0323: 0x0C8C000000000000,
		0x0324: 0x6C90000000000000,
		0x0325: 0x1C97256EEE460325,
		0x0326: 0x0C98000000000000,
		0x0327: 0x0C9C000000000000,
		0x0328: 0x0CA0000000000000,
		0x0329: 0x0CA4000000000000,
		0x032A: 0x0CA8000000000000,
		0x032B: 0x0CAC000000000000,
		0x032C: 0x0CB0000000000000,
		0x032D: 0x0CB4000000000000,
		0x032E: 0x0CB8000000000000,
		0x032F: 0x0CBC000000000000,
		0x0330: 0x0CC0000000000000,
		0x0331: 0x0CC4000000000000,
		0x0332: 0x0CC8000000000000,
		0x0333: 0x0CCC000000000000,
		0x0334: 0x1CD1B9A15E181334,
		0x0335: 0x4CD4000000000000,
		0x0336: 0x5CD8000000000000,
		0x0337: 0x0CDC000000000000,
		0x0338: 0x0CE0000000000000,
		0x0339: 0x0CE4000000000000,
		0x033A: 0x0CE8000000000000,
		0x0341: 0x1D0741358C181B41,
		0x0342: 0x0D08000000000000,
		0x0343: 0x0D0C000000000000,
		0x0344: 0x0D10000000000000,
		0x0345: 0x0D14000000000000,
		0x0346: 0x0D18000000000000,
		0x0347: 0x0D1C000000000000,
		0x0348: 0x0D20000000000000,
		0x0349: 0x0D24000000000000,
		0x034A: 0x0D28000000000000,
		0x034B: 0x0D2C000000000000,
		0x034C: 0x0D30000000000000,
		0x034D: 0x1D374D16FC18134D,
		0x034E: 0x0D38000000000000,
		0x034F: 0x0D3C000000000000,
		0x0350: 0x0D40000000000000,
		0x0351: 0x0D44000000000000,
		0x0352: 0x4D48000000000000,
		0x0353: 0x0D4C000000000000,
		0x0354: 0x0D50000000000000,
		0x0355: 0x0D54000000000000,
		0x0356: 0x0D58000000000000,
		0x0357: 0x1D5F572D98181357,
		0x0358: 0x0D60000000000000,
		0x0359: 0x0D64000000000000,
		0x035A: 0x2D68000000000000,
	}
}
