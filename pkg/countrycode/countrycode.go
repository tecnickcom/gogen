/*
Package countrycode provides information about countries and their ISO-3166 Codes.
The default data originates from multiple sources, including ISO-3166, CIA, United Nations, and Wikipedia.
The data is stored in a compact binary format for fast access and small memory footprint.
It is also possible to load custom country data.
*/
package countrycode

import "fmt"

// CountryData contains the country data to be returned.
type CountryData struct {
	StatusCode             uint8  `json:"statusCode"`
	Status                 string `json:"status"`
	Alpha2Code             string `json:"alpha2Code"`
	Alpha3Code             string `json:"alpha3Code"`
	NumericCode            string `json:"numericCode"`
	NameEnglish            string `json:"nameEnglish"`
	NameFrench             string `json:"nameFrench"`
	Region                 string `json:"region"`
	SubRegion              string `json:"subRegion"`
	IntermediateRegion     string `json:"intermediateRegion"`
	RegionCode             string `json:"regionCode"`
	SubRegionCode          string `json:"subRegionCode"`
	IntermediateRegionCode string `json:"intermediateRegionCode"`
	TLD                    string `json:"tld"`
}

// countryByAlpha2ID returns the country data for the given alpha-2 internal ID.
func (d *Data) countryByAlpha2ID(a2 uint16) (*CountryData, error) {
	ck, err := d.countryKeyByAlpha2ID(a2)
	if err != nil {
		return nil, err
	}

	el := decodeCountryKey(ck)

	status, err := d.statusByID(int(el.status))
	if err != nil {
		return nil, err
	}

	cd := &CountryData{
		StatusCode: el.status,
		Status:     status.name,
		Alpha2Code: decodeAlpha2(el.alpha2),
	}

	if el.alpha3 > 0 {
		cd.Alpha3Code = decodeAlpha3(el.alpha3)
		cd.NumericCode = fmt.Sprintf("%03d", el.numeric)

		name, err := d.countryNamesByAlpha2ID(el.alpha2)
		if err != nil {
			return nil, err
		}

		cd.NameEnglish = name.EN
		cd.NameFrench = name.FR

		region, err := d.regionByID(int(el.region))
		if err != nil {
			return nil, err
		}

		cd.RegionCode = region.code
		cd.Region = region.name

		subregion, err := d.subRegionByID(int(el.subregion))
		if err != nil {
			return nil, err
		}

		cd.SubRegionCode = subregion.code
		cd.SubRegion = subregion.name

		// no error check because el.intregion is max 3 bit and always valid
		intregion, _ := d.intermediateRegionByID(int(el.intregion))

		cd.IntermediateRegionCode = intregion.code
		cd.IntermediateRegion = intregion.name

		cd.TLD = decodeTLD(el.tld)
	}

	return cd, nil
}

// EnumStatus returns the status codes and names.
func (d *Data) EnumStatus() map[string]string {
	m := make(map[string]string, len(d.dStatusByID))

	for _, v := range d.dStatusByID {
		m[v.name] = v.code
	}

	return m
}

// EnumRegion returns the region codes and names.
func (d *Data) EnumRegion() map[string]string {
	m := make(map[string]string, len(d.dRegionByID))

	for _, v := range d.dRegionByID {
		m[v.name] = v.code
	}

	return m
}

// EnumSubRegion returns the sub-region codes and names.
func (d *Data) EnumSubRegion() map[string]string {
	m := make(map[string]string, len(d.dSubRegionByID))

	for _, v := range d.dSubRegionByID {
		m[v.name] = v.code
	}

	return m
}

// EnumIntermediateRegion returns the intermediate-region codes and names.
func (d *Data) EnumIntermediateRegion() map[string]string {
	m := make(map[string]string, len(d.dIntermediateRegionByID))

	for _, v := range d.dIntermediateRegionByID {
		m[v.name] = v.code
	}

	return m
}

// CountryByAlpha2Code returns the country data for the given ISO-3166 Alpha-2 code (e.g. "IT" for Italy).
func (d *Data) CountryByAlpha2Code(alpha2 string) (*CountryData, error) {
	a2, err := encodeAlpha2(alpha2)
	if err != nil {
		return nil, err
	}

	return d.countryByAlpha2ID(a2)
}

// CountryByAlpha3Code returns the country data for the given ISO-3166 Alpha-3 code (e.g. "ITA" for Italy).
func (d *Data) CountryByAlpha3Code(alpha3 string) (*CountryData, error) {
	a3, err := encodeAlpha3(alpha3)
	if err != nil {
		return nil, err
	}

	a2, err := d.alpha2IDByAlpha3ID(a3)
	if err != nil {
		return nil, err
	}

	return d.countryByAlpha2ID(a2)
}

// CountryByNumericCode returns the country data for the given ISO-3166 Numeric code (e.g. "380" for Italy).
func (d *Data) CountryByNumericCode(num string) (*CountryData, error) {
	nid, err := encodeNumeric(num)
	if err != nil {
		return nil, err
	}

	a2, err := d.alpha2IDByNumericID(nid)
	if err != nil {
		return nil, err
	}

	return d.countryByAlpha2ID(a2)
}

func (d *Data) countriesByAlpha2IDs(a2s []uint16) ([]*CountryData, error) {
	cds := make([]*CountryData, 0, len(a2s))

	for _, a2 := range a2s {
		cd, err := d.countryByAlpha2ID(a2)
		if err != nil {
			return nil, err
		}

		cds = append(cds, cd)
	}

	return cds, nil
}

// countriesByRegionID returns a list of countries in the given region internal ID (0-5).
// See EnumRegion() for a list of valid region IDs.
func (d *Data) countriesByRegionID(id uint8) ([]*CountryData, error) {
	a2s, err := d.alpha2IDsByRegionID(id)
	if err != nil {
		return nil, err
	}

	return d.countriesByAlpha2IDs(a2s)
}

// CountriesByRegionCode returns a list of countries in the given region code (e.g. "150" for Europe).
// See EnumRegion() for a list of valid region codes.
func (d *Data) CountriesByRegionCode(code string) ([]*CountryData, error) {
	id, err := d.regionIDByCode(code)
	if err != nil {
		return nil, err
	}

	return d.countriesByRegionID(id)
}

// CountriesByRegionName returns a list of countries in the given region name (e.g. "Europe").
// See EnumRegion() for a list of valid region names.
func (d *Data) CountriesByRegionName(name string) ([]*CountryData, error) {
	id, err := d.regionIDByName(name)
	if err != nil {
		return nil, err
	}

	return d.countriesByRegionID(id)
}

// countriesBySubRegionID returns a list of countries in the given sub-region internal ID (0-17).
// See EnumSubRegion() for a list of valid sub-region IDs.
func (d *Data) countriesBySubRegionID(id uint8) ([]*CountryData, error) {
	a2s, err := d.alpha2IDsBySubRegionID(id)
	if err != nil {
		return nil, err
	}

	return d.countriesByAlpha2IDs(a2s)
}

// CountriesBySubRegionCode returns a list of countries in the given sub-region code (e.g. "039" for Southern Europe).
// See EnumSubRegion() for a list of valid sub-region codes.
func (d *Data) CountriesBySubRegionCode(code string) ([]*CountryData, error) {
	id, err := d.subRegionIDByCode(code)
	if err != nil {
		return nil, err
	}

	return d.countriesBySubRegionID(id)
}

// CountriesBySubRegionName returns a list of countries in the given sub-region name (e.g. "Southern Europe").
// See EnumSubRegion() for a list of valid sub-region names.
func (d *Data) CountriesBySubRegionName(name string) ([]*CountryData, error) {
	id, err := d.subRegionIDByName(name)
	if err != nil {
		return nil, err
	}

	return d.countriesBySubRegionID(id)
}

// countriesByIntermediateRegionID returns a list of countries in the given intermediate region internal ID (0-7).
// See EnumIntermediateRegion() for a list of valid intermediate region IDs.
func (d *Data) countriesByIntermediateRegionID(id uint8) ([]*CountryData, error) {
	a2s, err := d.alpha2IDsByIntermediateRegionID(id)
	if err != nil {
		return nil, err
	}

	return d.countriesByAlpha2IDs(a2s)
}

// CountriesByIntermediateRegionCode returns a list of countries in the given intermediate region code (e.g. "014" for Eastern Africa).
// See EnumIntermediateRegion() for a list of valid intermediate region codes.
func (d *Data) CountriesByIntermediateRegionCode(code string) ([]*CountryData, error) {
	id, err := d.intermediateRegionIDByCode(code)
	if err != nil {
		return nil, err
	}

	return d.countriesByIntermediateRegionID(id)
}

// CountriesByIntermediateRegionName returns a list of countries in the given intermediate region name (e.g. "Eastern Africa").
// See EnumIntermediateRegion() for a list of valid intermediate region names.
func (d *Data) CountriesByIntermediateRegionName(name string) ([]*CountryData, error) {
	id, err := d.intermediateRegionIDByName(name)
	if err != nil {
		return nil, err
	}

	return d.countriesByIntermediateRegionID(id)
}

// CountriesByStatusID returns a list of countries with the given status ID (0-6).
// See EnumStatus() for a list of valid status IDs.
func (d *Data) CountriesByStatusID(id uint8) ([]*CountryData, error) {
	a2s, err := d.alpha2IDsByStatusID(id)
	if err != nil {
		return nil, err
	}

	return d.countriesByAlpha2IDs(a2s)
}

// CountriesByStatusName returns a list of countries with the given status name (e.g. "Officially assigned").
// See EnumStatus() for a list of valid status names.
func (d *Data) CountriesByStatusName(name string) ([]*CountryData, error) {
	id, err := d.statusIDByName(name)
	if err != nil {
		return nil, err
	}

	return d.CountriesByStatusID(id)
}

// CountriesByTLD returns a list of countries with the given TLD (top-level domain) code (e.g. "it" for Italy).
func (d *Data) CountriesByTLD(tld string) ([]*CountryData, error) {
	code, err := encodeTLD(tld)
	if err != nil {
		return nil, err
	}

	a2s, err := d.alpha2IDsByTLD(code)
	if err != nil {
		return nil, err
	}

	return d.countriesByAlpha2IDs(a2s)
}

// countryKey returns the internal binary representation for the given country data.
// This function can be used to rebuild the internal binary data from the exported data.
// It returns the internal Aplha2 ID and the CountryKey.
func (d *Data) countryKey(data *CountryData) (uint16, uint64, error) {
	status, err := d.statusIDByName(data.Status)
	if err != nil {
		return 0, 0, err
	}

	alpha2, err := encodeAlpha2(data.Alpha2Code)
	if err != nil {
		return 0, 0, err
	}

	alpha3, _ := encodeAlpha3(data.Alpha3Code)
	numeric, _ := encodeNumeric(data.NumericCode)
	region, _ := d.regionIDByName(data.Region)
	subregion, _ := d.subRegionIDByName(data.SubRegion)
	intregion, _ := d.intermediateRegionIDByName(data.IntermediateRegion)
	tld, _ := encodeTLD(data.TLD)

	ck := &countryKeyElem{
		status:    status,
		alpha2:    alpha2,
		alpha3:    alpha3,
		numeric:   numeric,
		region:    region,
		subregion: subregion,
		intregion: intregion,
		tld:       tld,
	}

	return ck.alpha2, ck.encodeCountryKey(), nil
}
