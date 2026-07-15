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
// so re-accepting it keeps passes consistent (idempotency); otherwise a
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

	// The {"name":"DB_PASSWORD","value":"secret"} shape (Kubernetes env, Docker
	// inspect, AWS tags): a label key whose string value names a secret marks
	// the sibling value member as the real secret; the key rule alone would
	// leave it visible. The cheap isJSONLabelKey guard keeps ordinary keys off
	// the (non-inlined) labeled-secret path.
	if isJSONLabelKey(key) {
		if next, redacted, ok := re.appendLabeledSecretJSON(src, i, key, valueStart, dst); ok {
			return next, redacted, true
		}
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

// appendLabeledSecretJSON handles the labeled-secret shape
// {"name":"<secret-name>", "value": <v>}: a label key (name/key/id, matched
// case-insensitively) whose quoted string value classifies as a sensitive key
// name, immediately followed by a sibling value member (value/val). It redacts
// the sibling value and returns the index past it. The label's own value is
// kept visible for debuggability unless the label key is itself sensitive (e.g.
// "key"), in which case it is redacted too, preserving the key rule's behavior.
//
// Only this forward order (label then value) is handled, the shape every
// producer of these maps emits. It stays convergent: on a re-pass the label
// value still names a secret and the sibling is already the inert marker.
func (re *Redactor) appendLabeledSecretJSON(src []byte, i int, labelKey []byte, labelValStart int, dst []byte) (int, []byte, bool) {
	if src[labelValStart] != '"' { // caller has already checked isJSONLabelKey(labelKey)
		return 0, dst, false
	}

	labelValEnd := parseJSONStringEnd(src, labelValStart)
	if labelValEnd <= labelValStart+1 || src[labelValEnd-1] != '"' {
		return 0, dst, false
	}

	// Classify the label value against the STRONG secret tokens only, not the
	// full key check: a `{"name":"DB_PASSWORD",...}` label marks the sibling as a
	// secret, but a weak/financial/PII label (`cardType`, `payment`, `account`)
	// must not blank ordinary {name,value} data (chart series, metrics, tags).
	if !re.isStrongSecretName(src[labelValStart+1 : labelValEnd-1]) {
		return 0, dst, false
	}

	sibValStart, sibValEnd, ok := findSiblingValueMember(src, labelValEnd)
	if !ok {
		return 0, dst, false
	}

	dst = append(dst, src[i:labelValStart]...) // `"name":` (label key + separator)

	if re.isSensitiveKey(labelKey) {
		dst = appendQuotedMarker(dst, re.marker)
	} else {
		dst = append(dst, src[labelValStart:labelValEnd]...) // keep `"DB_PASSWORD"`
	}

	dst = append(dst, src[labelValEnd:sibValStart]...) // `,"value":`
	dst = appendQuotedMarker(dst, re.marker)

	return sibValEnd, dst, true
}

// findSiblingValueMember, given labelValEnd just past a label's string value,
// requires the next member to be a value key (value/val) and returns the start
// and end of its value. It reports false for any other shape.
func findSiblingValueMember(src []byte, labelValEnd int) (int, int, bool) {
	j := skipJSONWhitespace(src, labelValEnd)
	if j >= len(src) || src[j] != ',' {
		return 0, 0, false
	}

	j = skipJSONWhitespace(src, j+1)
	if j >= len(src) || src[j] != '"' {
		return 0, 0, false
	}

	vq2, ok := findJSONStringClosingQuote(src, j+1)
	if !ok || !isJSONValueKey(src[j+1:vq2]) {
		return 0, 0, false
	}

	sibValStart, hasKV, done := findJSONValueStart(src, vq2)
	if done || !hasKV {
		return 0, 0, false
	}

	sibValEnd := jsonValueEnd(src, sibValStart)
	if sibValEnd == 0 {
		return 0, 0, false
	}

	return sibValStart, sibValEnd, true
}

// appendQuotedMarker appends a quoted redaction marker ("***").
func appendQuotedMarker(dst, marker []byte) []byte {
	dst = append(dst, '"')
	dst = append(dst, marker...)
	dst = append(dst, '"')

	return dst
}

// isJSONLabelKey reports whether key is one of the case-insensitive "label"
// keys of a {label, value} map (name/key/id).
func isJSONLabelKey(key []byte) bool {
	return equalsASCIIFold(key, "name") || equalsASCIIFold(key, "key") || equalsASCIIFold(key, "id")
}

// isJSONValueKey reports whether key is the case-insensitive value member of a
// {label, value} map (value/val).
func isJSONValueKey(key []byte) bool {
	return equalsASCIIFold(key, "value") || equalsASCIIFold(key, "val")
}

// appendRedactedEscapedJSONAt handles an escaped JSON key whose opening `\"`
// closes at src[i] (the '"'; the '\' at src[i-1] has already been emitted by the
// bulk copy). This is the shape produced when a JSON document is embedded as a
// string value inside another JSON document (captured request bodies, webhook
// payloads, audit records logged through a JSON handler). It redacts the escaped
// value of a sensitive escaped key:
//
//	{"body":"{\"password\":\"secret\"}"}  ->  {"body":"{\"password\":\"***\"}"}
//
// Only simple (backslash-free) escaped keys and escaped string values are
// handled; any inner escaping makes the handler bail, leaving the input
// unchanged. Bailing rather than guessing keeps the engine from ever
// mis-parsing a deeper escape level, and, because the emitted marker is
// backslash-free, keeps redaction convergent.
//
//nolint:gocyclo,cyclop // One guarded parse step per escaped-key/value component.
func (re *Redactor) appendRedactedEscapedJSONAt(src []byte, i int, dst []byte) (int, []byte, bool) {
	// The \" must open an object member: skipping whitespace backward from before
	// the '\' (raw spaces/tabs and the two-byte escaped \n/\t/\r a non-Go
	// serializer emits: Python json.dumps' ", "/": " separators, JS
	// JSON.stringify pretty-printing), the nearest structural byte is '{' or ','
	// (the outer bytes of the embedded object), or the start of input.
	if p := escapedJSONKeyContext(src, i-2); p >= 0 && src[p] != '{' && src[p] != ',' {
		return 0, dst, false
	}

	keyStart := i + 1

	keyClose, ok := simpleEscapedStringEnd(src, keyStart)
	if !ok {
		return 0, dst, false
	}

	// The key's closing \" is 2 bytes; then ':' (optional surrounding space),
	// then the value's opening \".
	j := skipJSONWhitespace(src, keyClose+2)
	if j >= len(src) || src[j] != ':' {
		return 0, dst, false
	}

	valStart := skipJSONWhitespace(src, j+1)
	if valStart+1 >= len(src) || src[valStart] != '\\' || src[valStart+1] != '"' {
		return 0, dst, false // only an escaped string value is handled
	}

	valClose, ok := simpleEscapedStringEnd(src, valStart+2)
	if !ok {
		return 0, dst, false
	}

	if !re.isSensitiveKey(src[keyStart:keyClose]) {
		return 0, dst, false
	}

	dst = append(dst, src[i:valStart]...) // `"password\":` (the '\' is already emitted)
	dst = append(dst, '\\', '"')
	dst = append(dst, re.marker...)
	dst = append(dst, '\\', '"')

	return valClose + 2, dst, true
}

// escapedJSONKeyContext scans backward from p over whitespace that can separate
// object members inside an escaped (embedded) JSON string: raw spaces/tabs and
// the two-byte escaped "\n"/"\t"/"\r" sequences (a '\' byte followed by
// 'n'/'t'/'r'). It returns the index of the nearest non-whitespace byte, or -1
// when the scan runs off the start of input (valid key context). This lets a
// non-Go serializer's spacing (Python json.dumps' ", "/": ", JS pretty-print)
// still reach the escaped-key parser instead of leaking every key after the
// first.
func escapedJSONKeyContext(src []byte, p int) int {
	for p >= 0 {
		switch {
		case src[p] == ' ' || src[p] == '\t':
			p--
		case (src[p] == 'n' || src[p] == 't' || src[p] == 'r') && p >= 1 && src[p-1] == '\\':
			p -= 2
		default:
			return p
		}
	}

	return -1
}

// simpleEscapedStringEnd, given p just past an opening `\"`, returns the index
// of the closing `\"`'s backslash when the content in between is simple: no
// backslash escapes, no bare quote, no line break. It reports false otherwise,
// so any inner escaping is left untouched rather than mis-parsed.
func simpleEscapedStringEnd(src []byte, p int) (int, bool) {
	for p+1 < len(src) {
		switch src[p] {
		case '\\':
			if src[p+1] == '"' {
				return p, true
			}

			return 0, false // a different escape: too complex, bail
		case '"', '\n', '\r':
			return 0, false
		}

		p++
	}

	return 0, false
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
