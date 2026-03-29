/*
Package redact provides lightweight, regex-based redaction utilities for
obscuring sensitive data before logging or debugging output is emitted.

# Problem

Operational logs and diagnostics often include raw HTTP headers, JSON payloads,
or URL-encoded form data. Without sanitization, these records can leak secrets
such as credentials, tokens, API keys, session identifiers, personal data, or
payment details. This package offers a fast, simple redaction pass that can be
applied at log boundaries.

# What It Redacts

[HTTPData] applies multiple redaction rules in sequence:

  - Authorization headers (`Authorization: ...`), preserving header name while
    replacing the value.
  - JSON key/value pairs where the key name contains secret-like substrings
    (authentication/session, crypto markers, legal-signing, financial, and PII
    keyword groups).
  - URL-encoded key/value pairs with matching sensitive key names.
  - Credit-card-like numeric patterns bounded by non-word separators.

All matched sensitive values are replaced with a constant marker so output
remains structurally useful while hiding private content.

# Key Features

  - Single-call redaction API for common HTTP-style payloads.
  - Broad keyword families to catch many real-world secret field names.
  - Preserves surrounding structure to keep logs searchable and debuggable.
  - No external dependencies beyond the standard regexp package.

# Usage

	safe := redact.HTTPData(rawHTTPPayload)
	logger.Info("request", "payload", safe)

# Important Notes

This package is best-effort pattern redaction, not a formal data-loss
prevention system. Always treat output as potentially sensitive and combine this
with least-privilege logging practices and structured logging controls.
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
	kwdFIN = `acc|amount|bal|bill|card|cc_|cv2|cvc|cvv|iban|pay|swift`

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

// HTTPData redacts sensitive HTTP-like data (Authorization headers, secret fields, and card patterns).
func HTTPData(s string) string {
	s = regexAuthorizationHeader.ReplaceAllString(s, redactAuthorizationHeader)
	s = regexJSONKey.ReplaceAllString(s, redactJSONKey)
	s = regexURLEncodedKey.ReplaceAllString(s, redactURLEncodedKey)
	s = regexCreditCard.ReplaceAllString(s, redactCreditCard)

	return s
}
