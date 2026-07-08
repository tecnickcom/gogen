package countrycode

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// FuzzAlpha2RoundTrip checks that any input accepted by encodeAlpha2 decodes
// back to the original string.
func FuzzAlpha2RoundTrip(f *testing.F) {
	for _, s := range []string{"IT", "ZW", "AA", "ZZ"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		code, err := encodeAlpha2(s)
		if err != nil {
			return
		}

		require.Equal(t, s, decodeAlpha2(code))
	})
}

// FuzzAlpha3RoundTrip checks that any input accepted by encodeAlpha3 decodes
// back to the original string.
func FuzzAlpha3RoundTrip(f *testing.F) {
	for _, s := range []string{"ITA", "ZWE", "AAA", "ZZZ"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		code, err := encodeAlpha3(s)
		if err != nil {
			return
		}

		require.Equal(t, s, decodeAlpha3(code))
	})
}

// FuzzTLDRoundTrip checks that any input accepted by encodeTLD decodes back to
// the original string.
func FuzzTLDRoundTrip(f *testing.F) {
	for _, s := range []string{"it", "zw", "aa", "zz"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		code, err := encodeTLD(s)
		if err != nil {
			return
		}

		require.Equal(t, s, decodeTLD(code))
	})
}

// FuzzCountryByAlpha2Code ensures the public lookup never panics on arbitrary
// input and only returns a nil result together with an error.
func FuzzCountryByAlpha2Code(f *testing.F) {
	data, err := New(nil)

	require.NoError(f, err)

	for _, s := range []string{"IT", "ZZ", "", "abc", "1!"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		got, err := data.CountryByAlpha2Code(s)
		if err != nil {
			require.Nil(t, got)
			return
		}

		require.NotNil(t, got)
	})
}
