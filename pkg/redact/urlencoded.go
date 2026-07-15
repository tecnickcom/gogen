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
// '#' is a boundary so a fragment parameter's key is isolated from the path
// before it: without it the backward scan for "...cb#access_token=" runs into
// the '/' of the path and the key is rejected, leaking OAuth implicit-flow
// tokens carried in a URL fragment.
//
// Invariant: this set must be a superset of isURLValueBoundary's: a redacted
// value is replaced by inert marker bytes, so any byte that can end a value
// must also fence the following key scan, or key extraction would differ
// between redaction passes.
func isURLKeyBoundary(c byte) bool {
	switch c {
	case '&', '?', '#', '\n', ' ', '\t', ';', ',', '"', '=', '<', '>', '\r':
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
	if bytes.IndexByte(rawKey, '/') >= 0 {
		return 0, dst, false
	}

	if re.isSensitiveKey(rawKey) {
		valueEnd := re.urlEncodedValueEnd(src, eq+1)

		dst = append(dst, '=')
		dst = append(dst, re.marker...)

		return valueEnd, dst, true
	}

	// HTML form-attribute labeled secret: a `name="<secret-name>"` attribute
	// followed by a sibling `value=`/`val=` attribute, the attribute analog of
	// the JSON {name,value} rule.
	if equalsASCIIFold(rawKey, "name") {
		return re.appendLabeledSecretAttr(src, eq, dst)
	}

	return 0, dst, false
}

// appendLabeledSecretAttr handles `<input name="password" value="secret">`:
// given eq at the '=' after a `name` attribute (verified by the caller) whose
// quoted value names a strong secret, it redacts the quoted value of a sibling
// `value`/`val` attribute on the same element while keeping the readable `name`
// attribute intact. Both quote styles are handled. It returns false for any
// other shape, so an ordinary `name=John` URL pair or a `name` attribute without
// a secret value is untouched. It stays convergent: on a re-pass the label still
// names a secret and the sibling is already the inert marker.
func (re *Redactor) appendLabeledSecretAttr(src []byte, eq int, dst []byte) (int, []byte, bool) {
	labelOpen := eq + 1
	if labelOpen >= len(src) || !isAttrQuote(src[labelOpen]) {
		return 0, dst, false
	}

	labelClose := attrQuotedValueEnd(src, labelOpen)
	if labelClose < 0 || !re.isStrongSecretName(src[labelOpen+1:labelClose]) {
		return 0, dst, false
	}

	sibOpen, sibClose, ok := findSiblingAttrValue(src, labelClose+1)
	if !ok {
		return 0, dst, false
	}

	dst = append(dst, src[eq:sibOpen+1]...) // `="password" value="` (through the sibling's opening quote)
	dst = append(dst, re.marker...)
	dst = append(dst, src[sibClose]) // the sibling's closing quote

	return sibClose + 1, dst, true
}

// attrQuotedValueEnd, given the opening quote at src[q], returns the index of
// the matching closing quote, or -1 when the value is unterminated or crosses a
// tag/line boundary (not a well-formed attribute value).
func attrQuotedValueEnd(src []byte, q int) int {
	quote := src[q]
	for j := q + 1; j < len(src); j++ {
		switch src[j] {
		case quote:
			return j
		case '\n', '\r', '<', '>':
			return -1
		}
	}

	return -1
}

// findSiblingAttrValue, given from just past a label attribute's closing quote,
// requires the next attribute to be `value`/`val` with a quoted value and
// returns the indexes of its opening and closing quotes.
func findSiblingAttrValue(src []byte, from int) (int, int, bool) {
	open, ok := siblingValueAttrQuote(src, from)
	if !ok {
		return 0, 0, false
	}

	closeQuote := attrQuotedValueEnd(src, open)
	if closeQuote < 0 {
		return 0, 0, false
	}

	return open, closeQuote, true
}

// siblingValueAttrQuote parses a whitespace-separated `value`/`val` attribute
// starting at from and returns the index of its opening quote. A whitespace
// separator between the two attributes is required.
func siblingValueAttrQuote(src []byte, from int) (int, bool) {
	j := skipInlineSpaces(src, from)
	if j == from {
		return 0, false // attributes must be whitespace-separated
	}

	nameStart := j
	for j < len(src) && isHeaderNameByte(src[j]) {
		j++
	}

	if !isAttrValueName(src[nameStart:j]) {
		return 0, false
	}

	j = skipInlineSpaces(src, j)
	if j >= len(src) || src[j] != '=' {
		return 0, false
	}

	j = skipInlineSpaces(src, j+1)
	if j >= len(src) || !isAttrQuote(src[j]) {
		return 0, false
	}

	return j, true
}

// isAttrQuote reports whether c opens an HTML attribute value (either quote style).
func isAttrQuote(c byte) bool {
	return c == '"' || c == '\''
}

// isAttrValueName reports whether an attribute name is the `value`/`val` member
// of a {name, value} attribute pair.
func isAttrValueName(name []byte) bool {
	return equalsASCIIFold(name, "value") || equalsASCIIFold(name, "val")
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
// redaction idempotent: a quoted value redacted on a first pass
// ("password=*** tail") must not have its trailing text consumed on a second
// pass. It returns 0 when the value is not a lone marker.
//
// The marker is therefore assumed to be inert text, and an already-redacted
// value is indistinguishable from an input value that happens to start with
// the marker plus a separator: "password=*** rest" keeps " rest" visible.
// Requiring the marker to span the whole value instead would fix that, but it
// would also make the second pass swallow the remainder of a logfmt line
// (`password=*** user=bob`, the first-pass output for a quoted secret), which
// is a far worse trade: real URL-encoded values encode spaces as %20/'+', so a
// value literally starting with "*** " is not a shape secrets take.
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
