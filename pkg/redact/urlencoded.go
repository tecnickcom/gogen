package redact

import (
	"bytes"
)

// isURLKeyBoundary reports whether c terminates the backward scan for a
// URL-encoded key. The key is bounded by parameter/pair separators and by
// whitespace, so a sensitive word earlier in a prose or log line (e.g. "email
// us at ... ref=123") cannot bleed into the key of an unrelated pair. Header
// lines with sensitive names are fully handled by the header rule instead.
// '=' itself is a boundary: keys never contain it, and without it a trailing
// "=" after a redacted pair ("token=***=") would re-match the sensitive key
// through the marker on a second pass, breaking idempotency.
//
// Invariant: this set must be a superset of isURLValueBoundary's — a redacted
// value is replaced by inert marker bytes, so any byte that can end a value
// must also fence the following key scan, or key extraction would differ
// between redaction passes.
func isURLKeyBoundary(c byte) bool {
	switch c {
	case '&', '?', '\n', ' ', '\t', ';', ',', '"', '=', '<', '>', '\r':
		return true
	default:
		return false
	}
}

// appendRedactedURLEncodedValueAt handles the '=' at src[eq]: when the key
// immediately before it is sensitive, it appends "=" and the redaction marker
// to dst, returning the index just past the value.
func (re *Redactor) appendRedactedURLEncodedValueAt(src []byte, eq int, dst []byte) (int, []byte, bool) {
	keyStart := eq
	for k := eq - 1; k >= 0; k-- {
		if isURLKeyBoundary(src[k]) {
			keyStart = k + 1

			break
		}

		if k == 0 {
			keyStart = 0
		}
	}

	rawKey := src[keyStart:eq]
	if bytes.IndexByte(rawKey, '/') >= 0 || !re.isSensitiveKey(rawKey) {
		return 0, dst, false
	}

	valueEnd := re.urlEncodedValueEnd(src, eq+1)

	dst = append(dst, '=')
	dst = append(dst, re.marker...)

	return valueEnd, dst, true
}

// urlEncodedValueEnd returns the end of a URL-encoded value, stopping at the
// next parameter separator ('&'), a double quote, or the line end ('\r'/'\n').
// A value that itself starts with a double quote (e.g. `password="a b"`) is
// instead consumed through its closing quote so the whole quoted secret is
// replaced.
//
// It intentionally does NOT stop at a space: a raw space in a sensitive value
// (e.g. "password=a b c") would otherwise leave everything after the first space
// visible. Consuming to the separator/line end keeps the whole value redacted,
// matching how JSON string values are handled, at the cost of over-redacting the
// tail of a request line when a sensitive param is last (safe over-redaction).
// The quote stop keeps a `key=value` pair embedded in a JSON string from eating
// the string's closing quote and the rest of the document.
func (re *Redactor) urlEncodedValueEnd(src []byte, i int) int {
	if i < len(src) && src[i] == '"' {
		return quotedValueEnd(src, i)
	}

	if end := re.redactedMarkerEnd(src, i); end > 0 {
		return end
	}

	for i < len(src) && !isURLValueBoundary(src[i]) {
		i++
	}

	return i
}

// isURLValueBoundary reports whether c terminates an unquoted URL-encoded
// value: the next parameter ('&'), a quote, pair separators, or the line end.
// '<' and '>' also stop the value so an unquoted XML attribute
// (`<user password=abc>`) does not consume the tag close and the rest of the
// document; genuine URL-encoded values carry %3C/%3E instead. ',' and ';'
// stop the value for the same reason they bound keys: consuming a separator
// would change the structural context of whatever follows (e.g. a JSON key's
// preceding comma), making a second redaction pass see a different document
// (idempotency).
func isURLValueBoundary(c byte) bool {
	switch c {
	case '&', '"', '\r', '\n', '<', '>', ',', ';':
		return true
	default:
		return false
	}
}

// redactedMarkerEnd reports the end of an already-redacted value: a marker
// followed by a non-word boundary spans only the marker itself, keeping
// redaction idempotent — a quoted value redacted on a first pass
// ("password=*** tail") must not have its trailing text consumed on a second
// pass. It returns 0 when the value is not a lone marker.
func (re *Redactor) redactedMarkerEnd(src []byte, i int) int {
	end := i + len(re.marker)

	if !re.markerAt(src, i) || (end < len(src) && isWordChar(src[end])) {
		return 0
	}

	return end
}

// quotedValueEnd returns the index just past the closing quote of a
// double-quoted value starting at src[i], or the line/input end when the quote
// is unterminated. Word characters glued directly to the closing quote are
// consumed as part of the value: leaving them would emit "***<char>", which a
// second redaction pass would then swallow differently (idempotency).
func quotedValueEnd(src []byte, i int) int {
	j := i + 1
	for j < len(src) && src[j] != '"' && src[j] != '\r' && src[j] != '\n' {
		j++
	}

	if j < len(src) && src[j] == '"' {
		j++
		for j < len(src) && isWordChar(src[j]) {
			j++
		}
	}

	return j
}
