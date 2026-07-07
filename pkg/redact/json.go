package redact

import (
	"bytes"
)

// likelyJSONKeyStart reports whether the double quote at src[i] plausibly
// opens a JSON object key: looking backwards, the nearest non-whitespace
// character must be '{' or ',' (or the start of input). String values and
// array elements are rejected so their contents are never mistaken for keys.
//
// A redaction marker directly before the quote also counts as key context:
// whatever stood there originally allowed a key match on the previous pass,
// so re-accepting it keeps passes consistent (idempotency) — otherwise a
// rewritten prefix would expose the key's interior to the other rules on the
// second pass.
func (re *Redactor) likelyJSONKeyStart(src []byte, i int) bool {
	for j := i - 1; j >= 0; j-- {
		switch src[j] {
		case ' ', '\t', '\n', '\r':
			continue
		case '{', ',':
			return true
		default:
			return re.markerEndsAt(src, j)
		}
	}

	return true
}

// appendRedactedSensitiveJSONAt handles a JSON key starting at the quote
// src[i]: when the key is sensitive and followed by a parseable value, it
// appends the key, its original separator, and the quoted redaction marker to
// dst, returning the index just past the value.
func (re *Redactor) appendRedactedSensitiveJSONAt(src []byte, i int, dst []byte) (int, []byte, bool) {
	q2, ok := findJSONStringClosingQuote(src, i+1)
	if !ok {
		return 0, dst, false
	}

	key := src[i+1 : q2]

	// Keys containing '=' are left to the URL-encoded rule, which redacts
	// '='-shaped pairs even inside quotes. Splitting ownership per pass (URL
	// rule first, JSON rule once the value looks well-formed) would rewrite
	// the same span differently on a second pass, breaking idempotency.
	if bytes.IndexByte(key, '=') >= 0 {
		return 0, dst, false
	}

	valueStart, hasKV, done := findJSONValueStart(src, q2)
	if done || !hasKV {
		return 0, dst, false
	}

	// Check key sensitivity before parsing the value: non-sensitive keys (the
	// common case) skip value scanning entirely, including balanced-container
	// scans for object/array values.
	if !re.isSensitiveKey(key) {
		return 0, dst, false
	}

	valueEnd := jsonValueEnd(src, valueStart)
	if valueEnd == 0 {
		return 0, dst, false
	}

	sep := src[q2+1 : valueStart]

	dst = append(dst, '"')
	dst = append(dst, key...)
	dst = append(dst, '"')
	dst = append(dst, sep...)
	dst = append(dst, '"')
	dst = append(dst, re.marker...)
	dst = append(dst, '"')

	return valueEnd, dst, true
}

// findJSONStringClosingQuote locates the closing quote of a JSON key starting
// after src[i-1]. Raw line breaks cannot appear inside a JSON string, so the
// scan treats them as unterminated: without this, quotes re-paired across
// lines by other rules' rewrites could form phantom multi-line "keys" and
// break idempotency.
func findJSONStringClosingQuote(src []byte, i int) (int, bool) {
	for i < len(src) {
		switch src[i] {
		case '\\':
			i += 2

			continue
		case '"':
			return i, true
		case '\n', '\r':
			return 0, false
		}

		i++
	}

	return 0, false
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

	if src[j] == '{' || src[j] == '[' {
		return parseJSONContainerEnd(src, j)
	}

	if literalEnd := jsonLiteralEnd(src, j); literalEnd > 0 {
		return delimitedValueEnd(src, literalEnd)
	}

	if src[j] == '-' || (src[j] >= '0' && src[j] <= '9') {
		return delimitedValueEnd(src, parseJSONNumberEnd(src, j))
	}

	return 0
}

// delimitedValueEnd validates that an unquoted JSON value (literal or number)
// ending at end is followed by a value delimiter, so malformed input like
// {"cvv":truex} is treated as unparseable instead of being spliced around the
// marker.
func delimitedValueEnd(src []byte, end int) int {
	if end < len(src) && !isJSONValueDelimiter(src[end]) {
		return 0
	}

	return end
}

func isJSONValueDelimiter(c byte) bool {
	switch c {
	case ',', '}', ']', ' ', '\t', '\n', '\r':
		return true
	default:
		return false
	}
}

// parseJSONContainerEnd returns the index just past a balanced JSON object or
// array beginning at src[j] (which must be '{' or '['), honoring string
// literals and escapes so that a delimiter inside a string does not end the
// span early. A container that is not balanced before end-of-input consumes
// everything to the end: truncated documents (cut log lines) hide the whole
// remainder instead of leaking non-sensitive-keyed fragments, and rewrites
// inside ambiguous quote soup cannot flip the balance between redaction
// passes (idempotency).
func parseJSONContainerEnd(src []byte, j int) int {
	depth := 0

	for j < len(src) {
		switch src[j] {
		case '{', '[':
			depth++
		case '}', ']':
			depth--
			if depth == 0 {
				return j + 1
			}
		case '"':
			j = parseJSONStringEnd(src, j)

			continue
		}

		j++
	}

	return len(src)
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
