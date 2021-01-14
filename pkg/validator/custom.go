package validator

import (
	"reflect"
	"regexp"

	vt "github.com/go-playground/validator/v10"
)

const (
	regexPatternE164      = `^[+]?[1-9][0-9]{7,14}$`
	regexPatternEIN       = `^[0-9]{2}-?[0-9]{7}$`
	regexPatternUSZIPCode = `^[0-9]{5}(?:-[0-9]{4})?$`
)

var (
	regexEIN       = regexp.MustCompile(regexPatternEIN)
	regexE164      = regexp.MustCompile(regexPatternE164)
	regexUSZIPCode = regexp.MustCompile(regexPatternUSZIPCode)
)

// CustomValidationTags maps custom tags with validation function
var CustomValidationTags = map[string]vt.Func{
	"e164":    isE164,
	"ein":     isEIN,
	"zipcode": isUSZIPCode,
	"usstate": isUSState,
}

// isE164 checks if the fields value is a valid E.164 phone number format (+123456789012345)
func isE164(fl vt.FieldLevel) bool {
	field := fl.Field()
	return regexE164.MatchString(field.String())
}

// isEIN checks if the fields value is a valid EIN US tax code (xx-xxxxxxx or xxxxxxxxx)
func isEIN(fl vt.FieldLevel) bool {
	field := fl.Field()
	return regexEIN.MatchString(field.String())
}

// isUSZIPCode checks if the fields value is a valid US ZIP code (xxxxx or xxxxx-xxxx)
func isUSZIPCode(fl vt.FieldLevel) bool {
	field := fl.Field()
	return regexUSZIPCode.MatchString(field.String())
}

// isUSState checks if the fields value is a valid 2-letter US state
func isUSState(fl vt.FieldLevel) bool {
	field := fl.Field()
	if field.Kind() == reflect.String {
		switch field.String() {
		case "AL", "AK", "AZ", "AR", "CA", "CO", "CT", "DE", "FL", "GA", "HI", "ID", "IL", "IN", "IA", "KS", "KY", "LA", "ME", "MD", "MA", "MI", "MN", "MS", "MO", "MT", "NE", "NV", "NH", "NJ", "NM", "NY", "NC", "ND", "OH", "OK", "OR", "PA", "RI", "SC", "SD", "TN", "TX", "UT", "VT", "VA", "WA", "WV", "WI", "WY":
			return true
		}
	}
	return false
}
