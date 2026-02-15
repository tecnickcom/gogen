/*
Package redact contains utility functions to obscure sensitive data.

This package is useful for logging and debugging, where sensitive data should
not be exposed.
*/
package redact

import (
	"regexp"
)

// Redaction patterns and replacements.
const (
	// Replacement string to hide the original information.
	redacted = `@~REDACTED~@`

	// Hide the Authorization header information.
	regexPatternAuthorizationHeader = `(?i)(authorization[\s]*:[\s]*).*`
	redactAuthorizationHeader       = `$1` + redacted

	// Hide common secret fields.
	regexPatternJSONKey = `(?i)"([^"]*)(key|password|secret|token)([^"]*)"([\s]*:[\s]*)"[^"]*"`
	redactJSONKey       = `"$1$2$3"$4"` + redacted + `"`

	regexPatternURLEncodedKey = `(?i)([^=&\n]*)(key|password|secret|token)([^=]*)=[^=&\n]*`
	redactURLEncodedKey       = `$1$2$3=` + redacted

	// General regular expression used to match Credit Card Numbers (Visa, MasterCard, American Express, Diners Club, Discover, and JCB cards).
	//
	//nolint:gosec
	regexPatternCreditCard = `([^\w\s]+)(?:4[0-9]{12}(?:[0-9]{3})?|[25][1-7][0-9]{14}|6(?:011|5[0-9][0-9])[0-9]{12}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|(?:2131|1800|35\d{3})\d{11})([^\w\s]+)`
	redactCreditCard       = `$1` + redacted + `$2`
)

// Compiled regular expressions.
var (
	regexAuthorizationHeader = regexp.MustCompile(regexPatternAuthorizationHeader)
	regexJSONKey             = regexp.MustCompile(regexPatternJSONKey)
	regexURLEncodedKey       = regexp.MustCompile(regexPatternURLEncodedKey)
	regexCreditCard          = regexp.MustCompile(regexPatternCreditCard)
)

// HTTPData returns the input string with sensitive HTTP data obscured.
// Redacts the Authorization header, password and key fields.
func HTTPData(s string) string {
	s = regexAuthorizationHeader.ReplaceAllString(s, redactAuthorizationHeader)
	s = regexJSONKey.ReplaceAllString(s, redactJSONKey)
	s = regexURLEncodedKey.ReplaceAllString(s, redactURLEncodedKey)
	s = regexCreditCard.ReplaceAllString(s, redactCreditCard)

	return s
}
