package countrycode_test

import (
	"testing"

	"github.com/tecnickcom/gogen/pkg/countrycode"
)

func BenchmarkNew(b *testing.B) {
	for b.Loop() {
		_, _ = countrycode.New(nil)
	}
}

func BenchmarkCountryByAlpha2Code(b *testing.B) {
	data, _ := countrycode.New(nil)

	b.ResetTimer()

	for b.Loop() {
		_, _ = data.CountryByAlpha2Code("IT")
	}
}

func BenchmarkCountryByAlpha3Code(b *testing.B) {
	data, _ := countrycode.New(nil)

	b.ResetTimer()

	for b.Loop() {
		_, _ = data.CountryByAlpha3Code("ITA")
	}
}

func BenchmarkCountryByNumericCode(b *testing.B) {
	data, _ := countrycode.New(nil)

	b.ResetTimer()

	for b.Loop() {
		_, _ = data.CountryByNumericCode("380")
	}
}

func BenchmarkCountriesByRegionCode(b *testing.B) {
	data, _ := countrycode.New(nil)

	b.ResetTimer()

	for b.Loop() {
		_, _ = data.CountriesByRegionCode("150")
	}
}
