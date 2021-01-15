package validator

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"

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
	"falseif": isFalseIf,
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

// isFalseIf is a special tag to be used in "OR" combination with another tag.
// It returns false if the specified parameter exist and has the specified value.
// This tag should never be used alone.
// The combined tag will be checked only if this validator returns false.
// Examples:
//     "falseif=Country US|usstate" checks if the field is a valid US state only if the Country field is set to "US".
//     "falseif=Country|usstate" checks if the field is a valid US state only if the Country field is set and not empty.
func isFalseIf(fl vt.FieldLevel) bool {
	param := strings.TrimSpace(fl.Param())
	if param == "" {
		return true
	}
	params := strings.SplitN(param, " ", 3)
	paramField, paramKind, nullable, found := fl.GetStructFieldOKAdvanced2(fl.Parent(), params[0])
	if !found {
		// the field in the param do not exist
		return true
	}
	if len(params) == 1 {
		return hasDefaultValue(paramField, paramKind, nullable)
	}
	return hasNotValue(paramField, paramKind, params[1])
}

// hasDefaultvalue returns true if the field has a default value (nil/zero) or if is unset/invalid.
func hasDefaultValue(value reflect.Value, kind reflect.Kind, nullable bool) bool {
	switch kind {
	case reflect.Invalid:
		return true
	case reflect.Slice, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Chan, reflect.Func:
		return value.IsNil()
	}
	return (nullable && value.Interface() == nil) || !value.IsValid() || (value.Interface() == reflect.Zero(value.Type()).Interface())
}

// hasNotValue returns true if the field has not the specified value.
// nolint:gocognit,gocyclo
func hasNotValue(value reflect.Value, kind reflect.Kind, paramValue string) bool {
	switch kind {
	case reflect.String:
		return value.String() != paramValue
	case reflect.Slice, reflect.Map, reflect.Array:
		p, err := strconv.ParseInt(paramValue, 0, 64)
		return err != nil || int64(value.Len()) != p
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p, err := strconv.ParseInt(paramValue, 0, 64)
		return err != nil || value.Int() != p
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p, err := strconv.ParseUint(paramValue, 0, 64)
		return err != nil || value.Uint() != p
	case reflect.Float32, reflect.Float64:
		p, err := strconv.ParseFloat(paramValue, 64)
		return err != nil || value.Float() != p
	case reflect.Bool:
		p, err := strconv.ParseBool(paramValue)
		return err != nil || value.Bool() != p
	}
	return true
}
