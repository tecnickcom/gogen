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
	"strings"
	"sync"
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

	redactionBufferPool = sync.Pool{ //nolint:gochecknoglobals
		New: func() any {
			b := make([]byte, 0, 1024)

			return &b
		},
	}
)

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

func nextLineBytes(src []byte) ([]byte, []byte) {
	nl := bytes.IndexByte(src, '\n')
	if nl < 0 {
		return src, nil
	}

	return src[:nl+1], src[nl+1:]
}

func redactAuthorizationLine(line []byte) ([]byte, bool) {
	if !hasAuthorizationPrefix(line) {
		return nil, false
	}

	colon := bytes.IndexByte(line[len(authorizationPrefix):], ':')
	if colon < 0 {
		return nil, false
	}

	headerEnd := len(authorizationPrefix) + colon + 1
	valueStart := skipInlineSpaces(line, headerEnd)
	trail := trailingNewline(line)

	buf := bytes.Buffer{}
	buf.Grow(len(line) + len(redactedBytes))
	buf.Write(line[:valueStart])
	buf.Write(redactedBytes)
	buf.Write(trail)

	return buf.Bytes(), true
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

func trailingNewline(line []byte) []byte {
	if len(line) > 0 && line[len(line)-1] == '\n' {
		return line[len(line)-1:]
	}

	return nil
}

// redactJSONKeys replaces values of sensitive JSON keys.
func redactJSONKeys(src []byte) []byte {
	buf := bytes.Buffer{}
	buf.Grow(len(src))

	i := 0

	for i < len(src) {
		nextIndex, done := redactNextJSONKeyValue(src, i, &buf)
		i = nextIndex

		if done {
			break
		}
	}

	return buf.Bytes()
}

func redactNextJSONKeyValue(src []byte, i int, buf *bytes.Buffer) (int, bool) {
	q1 := bytes.IndexByte(src[i:], '"')
	if q1 < 0 {
		buf.Write(src[i:])
		return len(src), true
	}

	q1 += i
	buf.Write(src[i:q1])

	q2 := bytes.IndexByte(src[q1+1:], '"')
	if q2 < 0 {
		buf.Write(src[q1:])
		return len(src), true
	}

	q2 += q1 + 1
	key := src[q1+1 : q2]

	valueStart, hasKV, done := findJSONValueStart(src, q2)
	if done {
		buf.Write(src[q1:])
		return len(src), true
	}

	if !hasKV {
		emitQuotedJSONToken(buf, key)
		return q2 + 1, false
	}

	valueEnd := jsonValueEnd(src, valueStart)
	if valueEnd == 0 {
		emitQuotedJSONToken(buf, key)
		return q2 + 1, false
	}

	sep := src[q2+1 : valueStart]

	if !isSensitiveKeyBytes(key) {
		emitQuotedJSONToken(buf, key)
		buf.Write(sep)
		buf.Write(src[valueStart:valueEnd])

		return valueEnd, false
	}

	// Keep JSON valid while replacing any primitive with a redacted string marker.
	emitQuotedJSONToken(buf, key)
	buf.Write(sep)
	buf.WriteByte('"')
	buf.Write(redactedBytes)
	buf.WriteByte('"')

	return valueEnd, false
}

func emitQuotedJSONToken(buf *bytes.Buffer, key []byte) {
	buf.WriteByte('"')
	buf.Write(key)
	buf.WriteByte('"')
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

//nolint:cyclop,gocognit,gocyclo
func redactCreditCards(src []byte) []byte {
	buf := bytes.Buffer{}
	buf.Grow(len(src))

	for i := 0; i < len(src); {
		if src[i] < '0' || src[i] > '9' {
			buf.WriteByte(src[i])
			i++

			continue
		}

		if i > 0 && isWordChar(src[i-1]) {
			buf.WriteByte(src[i])
			i++

			continue
		}

		j := i
		for j < len(src) && src[j] >= '0' && src[j] <= '9' {
			j++
		}

		if j < len(src) && isWordChar(src[j]) {
			buf.Write(src[i:j])
			i = j

			continue
		}

		if matchesCardPattern(src[i:j]) {
			buf.Write(redactedBytes)

			i = j

			continue
		}

		buf.Write(src[i:j])
		i = j
	}

	return buf.Bytes()
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

// redactURLEncodedKeys replaces values of sensitive URL-encoded keys.
//
// It scans forward for '=' characters.  For each one it looks backward to find
// the key start (the character after the last '&', '?', or '\n', or the start
// of src) and forward to find the value end (the next '&' or '\n', or the end
// of src).
// All content between the previous position and the current key=value
// pair is emitted verbatim, preserving every separator exactly once.
func redactURLEncodedKeys(src []byte) []byte {
	buf := bytes.Buffer{}
	buf.Grow(len(src))

	i := 0

	for i < len(src) {
		eq := bytes.IndexByte(src[i:], '=')
		if eq < 0 {
			buf.Write(src[i:])
			break
		}

		eq += i
		keyStart := urlEncodedKeyStart(src, i, eq)

		rawKey := src[keyStart:eq]

		// If the key contains '/', it belongs to a URL path segment, not an
		// encoded field.
		if bytes.IndexByte(rawKey, '/') >= 0 {
			buf.Write(src[i : eq+1])
			i = eq + 1

			continue
		}

		valueEnd := urlEncodedValueEnd(src, eq+1)

		if !isSensitiveKeyBytes(rawKey) {
			buf.Write(src[i:valueEnd])
			i = valueEnd

			continue
		}

		// Sensitive: emit everything up to and including '=', then the marker.
		buf.Write(src[i : eq+1])
		buf.Write(redactedBytes)

		i = valueEnd
	}

	return buf.Bytes()
}

func urlEncodedKeyStart(src []byte, i, eq int) int {
	keyStart := i

	for k := eq - 1; k >= i; k-- {
		c := src[k]
		if c == '&' || c == '?' || c == '\n' {
			return k + 1
		}
	}

	return keyStart
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

// redactAllInSinglePass applies all redaction rules in one traversal and writes
// to an append-backed destination slice to minimize allocations.
//

func redactAllInSinglePass(src []byte) []byte {
	return redactAllInSinglePassInto(make([]byte, 0, len(src)), src)
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

		if matchesCardPattern(src[i:j]) {
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
