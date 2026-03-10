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

	// kwdSEC is a pattern for Authentication & Session keywords.
	kwdSEC = `auth|bearer|cert|cookie|cred|dsn|jwt|key|login|pass|pwd|seal|secret|secur|sess|sgn|sid|sig|token`

	// kwdCRY is a pattern for Cryptographic & API Markers keywords.
	kwdCRY = `checksum|dsa|ecdsa|fingerprint|hash|hmac|pkcs|proof|rsa|salt`

	// kwdLEG is a pattern for Legal & Document Signing keywords.
	kwdLEG = `attestation|autograph|endorse`

	// kwdFIN is a pattern for Financial keywords.
	kwdFIN = `acc|amount|bal|bill|card|cc_|cvv|iban|pay|swift`

	// kwdPII is a pattern for Identity & Personal Data (PII) keywords.
	kwdPII = `addr|birth|cell|dob|email|mail|phone|social|ssn|tax|tel`

	// kwdALL is a collection of keyword sub-strings to redact.
	kwdALL = kwdSEC + "|" + kwdCRY + "|" + kwdLEG + "|" + kwdFIN + "|" + kwdPII

	// Hide the Authorization header information.
	regexPatternAuthorizationHeader = `(?i)(authorization[\s]*:[\s]*).*`
	redactAuthorizationHeader       = `$1` + redacted

	// Hide common secret fields.
	regexPatternJSONKey = `(?i)"([^"]*)(` + kwdALL + `)([^"]*)"([\s]*:[\s]*)"[^"]*"`
	redactJSONKey       = `"$1$2$3"$4"` + redacted + `"`

	regexPatternURLEncodedKey = `(?i)([^=&\n]*)(` + kwdALL + `)([^=]*)=[^=&\n]*`
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
