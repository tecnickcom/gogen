package countrycode_test

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Vonage/gosrvlib/pkg/countrycode"
)

func ExampleData_CountryByAlpha2Code() {
	data, err := countrycode.New(nil)
	if err != nil {
		log.Fatal(err)
	}

	got, err := data.CountryByAlpha2Code("ZW")
	if err != nil {
		log.Fatal(err)
	}

	b, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(b))

	// Output:
	// {
	//   "statusCode": 1,
	//   "status": "Officially assigned",
	//   "alpha2Code": "ZW",
	//   "alpha3Code": "ZWE",
	//   "numericCode": "716",
	//   "nameEnglish": "Zimbabwe",
	//   "nameFrench": "Zimbabwe (le)",
	//   "region": "Africa",
	//   "subRegion": "Sub-Saharan Africa",
	//   "intermediateRegion": "Eastern Africa",
	//   "regionCode": "002",
	//   "subRegionCode": "202",
	//   "intermediateRegionCode": "014",
	//   "tld": "zw"
	// }
}

func ExampleData_CountryByAlpha3Code() {
	data, err := countrycode.New(nil)
	if err != nil {
		log.Fatal(err)
	}

	got, err := data.CountryByAlpha3Code("ZWE")
	if err != nil {
		log.Fatal(err)
	}

	b, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(b))

	// Output:
	// {
	//   "statusCode": 1,
	//   "status": "Officially assigned",
	//   "alpha2Code": "ZW",
	//   "alpha3Code": "ZWE",
	//   "numericCode": "716",
	//   "nameEnglish": "Zimbabwe",
	//   "nameFrench": "Zimbabwe (le)",
	//   "region": "Africa",
	//   "subRegion": "Sub-Saharan Africa",
	//   "intermediateRegion": "Eastern Africa",
	//   "regionCode": "002",
	//   "subRegionCode": "202",
	//   "intermediateRegionCode": "014",
	//   "tld": "zw"
	// }
}

func ExampleData_CountryByNumericCode() {
	data, err := countrycode.New(nil)
	if err != nil {
		log.Fatal(err)
	}

	got, err := data.CountryByNumericCode("716")
	if err != nil {
		log.Fatal(err)
	}

	b, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(b))

	// Output:
	// {
	//   "statusCode": 1,
	//   "status": "Officially assigned",
	//   "alpha2Code": "ZW",
	//   "alpha3Code": "ZWE",
	//   "numericCode": "716",
	//   "nameEnglish": "Zimbabwe",
	//   "nameFrench": "Zimbabwe (le)",
	//   "region": "Africa",
	//   "subRegion": "Sub-Saharan Africa",
	//   "intermediateRegion": "Eastern Africa",
	//   "regionCode": "002",
	//   "subRegionCode": "202",
	//   "intermediateRegionCode": "014",
	//   "tld": "zw"
	// }
}

//nolint:testableexamples
func ExampleData_CountryByAlpha2Code_export() {
	data, err := countrycode.New(nil)
	if err != nil {
		log.Fatal(err)
	}

	expCountries := make([]*countrycode.CountryData, 0, 26*26)

	// Generate all 2-letter country codes possible combinations, from AA to ZZ.
	for c1 := 'A'; c1 <= 'Z'; c1++ {
		for c0 := 'A'; c0 <= 'Z'; c0++ {
			alpha2 := string([]rune{c1, c0})

			// Decode country data from the 2-letter country code.
			country, err := data.CountryByAlpha2Code(alpha2)
			if err != nil {
				log.Fatal(err)
			}

			expCountries = append(expCountries, country)
		}
	}

	// Export the country data to JSON.
	jsonExpCountries, err := json.MarshalIndent(expCountries, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(jsonExpCountries))
}
