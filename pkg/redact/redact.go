/*
Package redact provides lightweight, pattern-based redaction utilities for
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

# Credit-Card Detection and the Optional Luhn Gate

By default, any 13-16 digit run that matches a known card prefix (Visa,
Mastercard, Amex, Discover, Diners, JCB, ...) and is bounded by non-word
characters is redacted. This is deliberate over-redaction: it is the safe
default and may also redact unrelated numeric identifiers that happen to share a
card prefix and length.

Callers that prefer fewer false positives can enable an additional Luhn-checksum
gate. When enabled, a digit run is only redacted if it matches a known prefix AND
passes the Luhn checksum:

	redact.SetLuhnCheck(true)  // opt in: prefix + Luhn must both pass.
	redact.SetLuhnCheck(false) // default: prefix match alone triggers redaction.

The setting is process-wide and concurrency-safe. It defaults to off so existing
output is unchanged. Enabling it may cause malformed or non-Luhn test numbers to
be left visible.

# Key Features

  - Single-call redaction API for common HTTP-style payloads.
  - Broad keyword families to catch many real-world secret field names.
  - Preserves surrounding structure to keep logs searchable and debuggable.
  - No external dependencies beyond the Go standard library.

# Usage

	safe := redact.HTTPData(rawHTTPPayload)
	logger.Info("request", "payload", safe)

For high-throughput paths, reuse an output buffer to avoid per-call allocations:

	var dst []byte
	for _, payload := range payloads {
		dst = redact.HTTPDataBytesInto(dst, payload)
		logger.Info("request", "payload", string(dst))
	}

# Important Notes

This package is best-effort pattern redaction, not a formal data-loss
prevention system. Always treat output as potentially sensitive and combine this
with least-privilege logging practices and structured logging controls.
*/
package redact

import (
	"bytes"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"
)

// Redaction patterns and replacements.
const (
	// RedactionMarker is the placeholder used to replace sensitive values.
	RedactionMarker = `***`
)

// Reusable byte constants.
var (
	redactedBytes = []byte(RedactionMarker) //nolint:gochecknoglobals

	// authorizationPrefix is the ASCII-lowercased header name used for the fast
	// case-insensitive Authorization header scan.
	authorizationPrefix = []byte("authorization") //nolint:gochecknoglobals

	redactionBufferPool = sync.Pool{New: newRedactionBuffer} //nolint:gochecknoglobals

	// luhnCheckEnabled gates the optional Luhn-checksum validation for
	// credit-card detection. It defaults to false (disabled) so the package's
	// out-of-the-box redaction output is unchanged: any 13-16 digit run matching
	// a known card prefix is redacted (deliberate over-redaction as a safe
	// default). When enabled, such a run is only redacted if it ALSO passes the
	// Luhn checksum, reducing false positives at the cost of missing malformed
	// or test card numbers.
	luhnCheckEnabled atomic.Bool //nolint:gochecknoglobals
)

// SetLuhnCheck enables or disables the optional Luhn-checksum gate for
// credit-card detection.
//
// This setting is process-wide and safe for concurrent use. It is additive and
// off by default: with the default (disabled) behavior, every 13-16 digit run
// matching a known card prefix is redacted (a deliberate, safe over-redaction).
// When enabled, only digit runs that match a known prefix AND pass the Luhn
// checksum are redacted, which reduces over-redaction of unrelated numeric
// identifiers at the cost of possibly missing malformed numbers.
func SetLuhnCheck(enabled bool) {
	luhnCheckEnabled.Store(enabled)
}

// LuhnCheckEnabled reports whether the optional Luhn-checksum gate for
// credit-card detection is currently enabled. It is safe for concurrent use.
func LuhnCheckEnabled() bool {
	return luhnCheckEnabled.Load()
}

// normalizeKey converts a key name to a tokenized lowercase form suitable for
// boundary-aware keyword checks (e.g., camelCase -> snake_case tokens).
func normalizeKey(s string) string {
	var b strings.Builder
	b.Grow(len(s) * 2)

	prevIsLowerOrDigit := false
	prevIsUnderscore := true

	for _, r := range s {
		prevIsLowerOrDigit, prevIsUnderscore = appendNormalizedRune(
			&b,
			r,
			prevIsLowerOrDigit,
			prevIsUnderscore,
		)
	}

	return strings.Trim(b.String(), "_")
}

func appendNormalizedRune(b *strings.Builder, r rune, prevIsLowerOrDigit, prevIsUnderscore bool) (bool, bool) {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		isUpper := unicode.IsUpper(r)
		if isUpper && prevIsLowerOrDigit && !prevIsUnderscore {
			b.WriteByte('_')
		}

		b.WriteRune(unicode.ToLower(r))

		return unicode.IsLower(r) || unicode.IsDigit(r), false
	}

	if !prevIsUnderscore {
		b.WriteByte('_')
	}

	return false, true
}

func hasAuthorizationPrefix(line []byte) bool {
	if len(line) <= len(authorizationPrefix) {
		return false
	}

	for i, c := range authorizationPrefix {
		if line[i]|0x20 != c { // ASCII lower-case trick.
			return false
		}
	}

	return true
}

func skipInlineSpaces(src []byte, i int) int {
	for i < len(src) && (src[i] == ' ' || src[i] == '\t') {
		i++
	}

	return i
}

func findJSONValueStart(src []byte, q2 int) (int, bool, bool) {
	j := skipJSONWhitespace(src, q2+1)
	if j >= len(src) {
		return 0, false, true
	}

	if src[j] != ':' {
		return 0, false, false
	}

	valueStart := skipJSONWhitespace(src, j+1)
	if valueStart >= len(src) {
		return 0, false, true
	}

	return valueStart, true, false
}

func skipJSONWhitespace(src []byte, i int) int {
	for i < len(src) && (src[i] == ' ' || src[i] == '\t' || src[i] == '\n' || src[i] == '\r') {
		i++
	}

	return i
}

func jsonValueEnd(src []byte, j int) int {
	if src[j] == '"' {
		return parseJSONStringEnd(src, j)
	}

	if literalEnd := jsonLiteralEnd(src, j); literalEnd > 0 {
		return literalEnd
	}

	if src[j] == '-' || (src[j] >= '0' && src[j] <= '9') {
		return parseJSONNumberEnd(src, j)
	}

	return 0
}

func jsonLiteralEnd(src []byte, j int) int {
	switch {
	case hasPrefixAt(src, j, "true"):
		return j + len("true")
	case hasPrefixAt(src, j, "false"):
		return j + len("false")
	case hasPrefixAt(src, j, "null"):
		return j + len("null")
	default:
		return 0
	}
}

func parseJSONStringEnd(src []byte, j int) int {
	vEnd := j + 1
	for vEnd < len(src) {
		if src[vEnd] == '\\' {
			vEnd += 2
			continue
		}

		if src[vEnd] == '"' {
			return vEnd + 1
		}

		vEnd++
	}

	return vEnd
}

func hasPrefixAt(src []byte, i int, lit string) bool {
	if i+len(lit) > len(src) {
		return false
	}

	for j := range len(lit) {
		if src[i+j] != lit[j] {
			return false
		}
	}

	return true
}

func parseJSONNumberEnd(src []byte, j int) int {
	vEnd := j
	for vEnd < len(src) {
		c := src[vEnd]
		if c == '-' || c == '+' || c == '.' || c == 'e' || c == 'E' || (c >= '0' && c <= '9') {
			vEnd++
			continue
		}

		break
	}

	return vEnd
}

func isWordChar(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

//nolint:cyclop,gocognit,gocyclo,nestif
func matchesCardPattern(digits []byte) bool {
	n := len(digits)
	if n < 13 || n > 16 {
		return false
	}

	if digits[0] == '4' && (n == 13 || n == 16) {
		return true
	}

	if n == 16 {
		if (digits[0] == '2' || digits[0] == '5') && digits[1] >= '1' && digits[1] <= '7' {
			return true
		}

		if digits[0] == '6' {
			if digits[1] == '0' && digits[2] == '1' && digits[3] == '1' {
				return true
			}

			if digits[1] == '5' && digits[2] >= '0' && digits[2] <= '9' && digits[3] >= '0' && digits[3] <= '9' {
				return true
			}
		}

		if digits[0] == '3' && digits[1] == '5' && digits[2] >= '0' && digits[2] <= '9' && digits[3] >= '0' && digits[3] <= '9' && digits[4] >= '0' && digits[4] <= '9' {
			return true
		}
	}

	if n == 15 {
		if digits[0] == '3' && (digits[1] == '4' || digits[1] == '7') {
			return true
		}

		if digits[0] == '2' && digits[1] == '1' && digits[2] == '3' && digits[3] == '1' {
			return true
		}

		if digits[0] == '1' && digits[1] == '8' && digits[2] == '0' && digits[3] == '0' {
			return true
		}
	}

	if n == 14 && digits[0] == '3' {
		if digits[1] == '0' && digits[2] >= '0' && digits[2] <= '5' {
			return true
		}

		if digits[1] == '6' || digits[1] == '8' {
			return true
		}
	}

	return false
}

// isCreditCard reports whether a run of ASCII digits should be redacted as a
// credit-card number. It always requires a known card prefix; when the optional
// Luhn gate is enabled it additionally requires a valid Luhn checksum.
func isCreditCard(digits []byte) bool {
	if !matchesCardPattern(digits) {
		return false
	}

	if luhnCheckEnabled.Load() {
		return passesLuhn(digits)
	}

	return true
}

// passesLuhn reports whether a run of ASCII digits satisfies the Luhn checksum.
func passesLuhn(digits []byte) bool {
	sum := 0
	double := false

	for _, c := range slices.Backward(digits) {
		d := int(c - '0')

		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}

		sum += d
		double = !double
	}

	return sum%10 == 0
}

func urlEncodedValueEnd(src []byte, i int) int {
	for i < len(src) {
		c := src[i]
		if c == '&' || c == '\n' {
			break
		}

		i++
	}

	return i
}

// redactAllInSinglePassInto applies all redaction rules while appending output
// into dst, which is reset to length 0 before use.
//
//nolint:cyclop,funlen,gocognit,gocyclo
func redactAllInSinglePassInto(dst, src []byte) []byte {
	dst = dst[:0]

	for i := 0; i < len(src); {
		if i == 0 || src[i-1] == '\n' {
			lineEnd := i
			for lineEnd < len(src) && src[lineEnd] != '\n' {
				lineEnd++
			}

			if lineEnd < len(src) {
				lineEnd++
			}

			line := src[i:lineEnd]
			if hasAuthorizationPrefix(line) {
				if redacted, ok := appendRedactedAuthorizationLine(dst, line); ok {
					dst = redacted
					i += len(line)

					continue
				}
			}
		}

		if src[i] == '"' && likelyJSONKeyStart(src, i) {
			nextIndex, redacted, ok := appendRedactedSensitiveJSONAt(src, i, dst)
			if ok {
				dst = redacted
				i = nextIndex

				continue
			}
		}

		if src[i] == '=' {
			valueEnd, redacted, ok := appendRedactedURLEncodedValueAt(src, i, dst)
			if ok {
				dst = redacted
				i = valueEnd

				continue
			}
		}

		if src[i] == '\n' {
			dst = append(dst, src[i])
			i++

			continue
		}

		if !isDigitByte(src[i]) {
			j := i + 1
			for j < len(src) {
				c := src[j]
				if c == '"' || c == '=' || c == '\n' || isDigitByte(c) {
					break
				}

				j++
			}

			dst = append(dst, src[i:j]...)
			i = j

			continue
		}

		if i > 0 && isWordChar(src[i-1]) {
			dst = append(dst, src[i])
			i++

			continue
		}

		j := i
		for j < len(src) && src[j] >= '0' && src[j] <= '9' {
			j++
		}

		if j < len(src) && isWordChar(src[j]) {
			dst = append(dst, src[i:j]...)
			i = j

			continue
		}

		if isCreditCard(src[i:j]) {
			dst = append(dst, redactedBytes...)
			i = j

			continue
		}

		dst = append(dst, src[i:j]...)
		i = j
	}

	return dst
}

func likelyJSONKeyStart(src []byte, i int) bool {
	for j := i - 1; j >= 0; j-- {
		switch src[j] {
		case ' ', '\t', '\n', '\r':
			continue
		case '{', ',':
			return true
		default:
			return false
		}
	}

	return true
}

func isDigitByte(c byte) bool {
	return c >= '0' && c <= '9'
}

func appendRedactedAuthorizationLine(dst, line []byte) ([]byte, bool) {
	colon := bytes.IndexByte(line[len(authorizationPrefix):], ':')
	if colon < 0 {
		return dst, false
	}

	headerEnd := len(authorizationPrefix) + colon + 1
	valueStart := skipInlineSpaces(line, headerEnd)

	dst = append(dst, line[:valueStart]...)
	dst = append(dst, redactedBytes...)

	if len(line) > 0 && line[len(line)-1] == '\n' {
		dst = append(dst, '\n')
	}

	return dst, true
}

func appendRedactedSensitiveJSONAt(src []byte, i int, dst []byte) (int, []byte, bool) {
	q2, ok := findJSONStringClosingQuote(src, i+1)
	if !ok {
		return 0, dst, false
	}

	key := src[i+1 : q2]

	valueStart, hasKV, done := findJSONValueStart(src, q2)
	if done || !hasKV {
		return 0, dst, false
	}

	valueEnd := jsonValueEnd(src, valueStart)
	if valueEnd == 0 || !isSensitiveKeyBytes(key) {
		return 0, dst, false
	}

	sep := src[q2+1 : valueStart]

	dst = append(dst, '"')
	dst = append(dst, key...)
	dst = append(dst, '"')
	dst = append(dst, sep...)
	dst = append(dst, '"')
	dst = append(dst, redactedBytes...)
	dst = append(dst, '"')

	return valueEnd, dst, true
}

func findJSONStringClosingQuote(src []byte, i int) (int, bool) {
	for i < len(src) {
		if src[i] == '\\' {
			i += 2

			continue
		}

		if src[i] == '"' {
			return i, true
		}

		i++
	}

	return 0, false
}

func appendRedactedURLEncodedValueAt(src []byte, eq int, dst []byte) (int, []byte, bool) {
	keyStart := eq
	for k := eq - 1; k >= 0; k-- {
		c := src[k]
		if c == '&' || c == '?' || c == '\n' {
			keyStart = k + 1

			break
		}

		if k == 0 {
			keyStart = 0
		}
	}

	rawKey := src[keyStart:eq]
	if bytes.IndexByte(rawKey, '/') >= 0 || !isSensitiveKeyBytes(rawKey) {
		return 0, dst, false
	}

	valueEnd := urlEncodedValueEnd(src, eq+1)

	dst = append(dst, '=')
	dst = append(dst, redactedBytes...)

	return valueEnd, dst, true
}

// HTTPDataBytes redacts sensitive HTTP-like data (Authorization headers, secret fields, and card patterns).
func HTTPDataBytes(b []byte) []byte {
	return redactAllInSinglePassInto(make([]byte, 0, len(b)), b)
}

// HTTPDataBytesInto redacts sensitive HTTP-like data and appends the result
// into dst (after resetting its length to zero), allowing callers to reuse
// output buffers across calls.
func HTTPDataBytesInto(dst, src []byte) []byte {
	return redactAllInSinglePassInto(dst, src)
}

// HTTPDataBytesPooled redacts sensitive HTTP-like data using an internal
// pooled buffer and passes the result to consume.
//
// The passed slice is only valid during the consume call and must not be
// retained after consume returns.
func HTTPDataBytesPooled(src []byte, consume func([]byte)) {
	if consume == nil {
		return
	}

	dst := getPooledRedactionBuffer(len(src))
	out := redactAllInSinglePassInto(dst, src)
	consume(out)
	putPooledRedactionBuffer(out)
}

// newRedactionBuffer is the sync.Pool factory for reusable output buffers.
func newRedactionBuffer() any {
	b := make([]byte, 0, 1024)

	return &b
}

func getPooledRedactionBuffer(minCap int) []byte {
	bp, _ := redactionBufferPool.Get().(*[]byte)
	if bp == nil {
		b := make([]byte, 0, minCap)

		return b
	}

	b := *bp
	if cap(b) < minCap {
		return make([]byte, 0, minCap)
	}

	return b[:0]
}

func putPooledRedactionBuffer(b []byte) {
	// Avoid keeping very large buffers indefinitely in the pool.
	const maxPooledCap = 1 << 20
	if cap(b) > maxPooledCap {
		return
	}

	b = b[:0]
	redactionBufferPool.Put(&b)
}

// HTTPDataString redacts sensitive HTTP-like data from a byte slice and returns the result as a string.
// It uses a pooled output buffer to reduce allocations and is the preferred form when the caller
// already holds a []byte (e.g. from httputil.DumpRequest / DumpResponse).
func HTTPDataString(b []byte) string {
	var out string

	HTTPDataBytesPooled(b, func(redacted []byte) {
		out = string(redacted)
	})

	return out
}

// HTTPData redacts sensitive HTTP-like data (Authorization headers, secret fields, and card patterns).
func HTTPData(s string) string {
	return string(HTTPDataBytes([]byte(s)))
}
