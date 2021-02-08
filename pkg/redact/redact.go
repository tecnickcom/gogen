// Package redact contains utilities functions to obscure sensitive data.
package redact

import (
	"regexp"
)

const (
	redacted = `@~REDACTED~@`

	regexPatternAuthorizationHeader = `(?i)(Authorization[\s]*:[\s]*).*`
	redactAuthorizationHeader       = `$1` + redacted

	regexPatternJSONKey = `(?i)"(.*)(password|key)"([\s]*:[\s]*)".*"`
	redactJSONKey       = `"$1$2"$3"` + redacted + `"`

	regexPatternURLEncodedKey = `(?i)([^=&\n]*)(password|key)=[^=&\n]*`
	redactURLEncodedKey       = `$1$2=` + redacted
)

var (
	regexAuthorizationHeader = regexp.MustCompile(regexPatternAuthorizationHeader)
	regexJSONKey             = regexp.MustCompile(regexPatternJSONKey)
	regexURLEncodedKey       = regexp.MustCompile(regexPatternURLEncodedKey)
)

// HTTPData returns the input string with sensitive HTTP data obscured.
// Redacts the Authorization header, password and key fields.
func HTTPData(s string) string {
	s = regexAuthorizationHeader.ReplaceAllString(s, redactAuthorizationHeader)
	s = regexJSONKey.ReplaceAllString(s, redactJSONKey)
	s = regexURLEncodedKey.ReplaceAllString(s, redactURLEncodedKey)

	return s
}
