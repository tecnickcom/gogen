package redact

import (
	"bytes"
)

// PEM boundary markers for private-key blocks.
const (
	pemBeginPrefix = "-----BEGIN "
	pemEndPrefix   = "-----END "
)

// pemPrivateKeyLabel is the label fragment identifying secret PEM blocks
// (RSA/EC/OPENSSH/ENCRYPTED/... PRIVATE KEY). Public material such as
// CERTIFICATE or PUBLIC KEY blocks is deliberately left visible.
var pemPrivateKeyLabel = []byte("PRIVATE KEY-----") //nolint:gochecknoglobals

// appendRedactedPEMKeyAt handles a '-' at a line start src[i]: when the line
// is a PEM "-----BEGIN ... PRIVATE KEY-----" boundary, the whole base64 body
// is replaced with a single marker line, preserving the BEGIN line (and its
// CRLF/LF style) for context. It returns the index of the matching
// "-----END" line so it is emitted verbatim by the main scan; an unterminated
// block is redacted through end of input (safe direction).
func (re *Redactor) appendRedactedPEMKeyAt(src []byte, i int, dst []byte) (int, []byte, bool) {
	if !hasPrefixAt(src, i, pemBeginPrefix) {
		return 0, dst, false
	}

	lineEnd := scanToLineFeed(src, i)

	line := src[i:lineEnd]
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}

	// The BEGIN marker must terminate the line: a line with content after the
	// label (e.g. a whole key blob with escaped "\n" sequences) is left to the
	// inline PEM rule, which redacts the blob itself instead of mistaking the
	// following unrelated lines for the key body.
	idx := bytes.Index(line, pemPrivateKeyLabel)
	if idx < 0 || idx+len(pemPrivateKeyLabel) != len(line) {
		return 0, dst, false
	}

	bodyStart := lineEnd
	if bodyStart < len(src) {
		bodyStart++ // consume the BEGIN line's '\n'.
	}

	endLine := findPEMEndLine(src, bodyStart)
	if endLine == bodyStart {
		// Empty body: nothing to hide; leave the block to the normal scan.
		return 0, dst, false
	}

	dst = append(dst, src[i:bodyStart]...) // BEGIN line incl. terminator.
	dst = append(dst, re.marker...)
	dst = appendPEMBodyTerminator(dst, src, endLine)

	return endLine, dst, true
}

// findPEMEndLine returns the start index of the first line at or after
// bodyStart that begins with the PEM END prefix, or len(src) when the block is
// unterminated.
func findPEMEndLine(src []byte, bodyStart int) int {
	endLine := bodyStart
	for endLine < len(src) && !hasPrefixAt(src, endLine, pemEndPrefix) {
		nl := scanToLineFeed(src, endLine)
		if nl >= len(src) {
			return len(src)
		}

		endLine = nl + 1
	}

	return endLine
}

// appendPEMBodyTerminator preserves the body's final line terminator so the
// END line stays on its own line with the block's original CRLF/LF style.
func appendPEMBodyTerminator(dst, src []byte, endLine int) []byte {
	if endLine == 0 || endLine > len(src) || src[endLine-1] != '\n' {
		return dst
	}

	if endLine >= 2 && src[endLine-2] == '\r' {
		return append(dst, '\r', '\n')
	}

	return append(dst, '\n')
}

// scanToLineFeed returns the index of the next '\n' at or after i, or len(src).
func scanToLineFeed(src []byte, i int) int {
	for i < len(src) && src[i] != '\n' {
		i++
	}

	return i
}

// pemEndMarker is the needle located to terminate an inline PEM body.
var pemEndMarker = []byte(pemEndPrefix) //nolint:gochecknoglobals

// appendRedactedInlinePEMKeyAt handles a '-' anywhere in the input: when it
// starts a "-----BEGIN ... PRIVATE KEY-----" marker, the key material after
// the marker is replaced up to the "-----END" marker (kept visible), or up to
// the closing quote / line end / EOF when unterminated. This catches keys
// embedded mid-line — most commonly a JSON string value carrying a whole PEM
// blob with escaped "\n" sequences under a non-sensitive key name.
func (re *Redactor) appendRedactedInlinePEMKeyAt(src []byte, i int, dst []byte) (int, []byte, bool) {
	if !hasPrefixAt(src, i, pemBeginPrefix) {
		return 0, dst, false
	}

	// The label must identify secret material and complete the BEGIN marker.
	lineEnd := scanToLineFeed(src, i)

	idx := bytes.Index(src[i:lineEnd], pemPrivateKeyLabel)
	if idx < 0 {
		return 0, dst, false
	}

	markerEnd := i + idx + len(pemPrivateKeyLabel)

	end := inlinePEMBodyEnd(src, markerEnd)
	if isWhitespaceOnly(src[markerEnd:end]) {
		// Empty (or whitespace-only) body: nothing to hide; a real key body is
		// base64 material. Leave the markers to the normal scan.
		return 0, dst, false
	}

	dst = append(dst, src[i:markerEnd]...)
	dst = append(dst, re.marker...)

	return end, dst, true
}

// isWhitespaceOnly reports whether b contains only whitespace bytes.
func isWhitespaceOnly(b []byte) bool {
	for _, c := range b {
		if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			return false
		}
	}

	return true
}

// inlinePEMBodyEnd returns the end of the key material following an inline
// BEGIN marker: the start of the "-----END" marker when present, else the
// closing '"' (JSON-embedded blob), else the line end or EOF.
func inlinePEMBodyEnd(src []byte, markerEnd int) int {
	if idx := bytes.Index(src[markerEnd:], pemEndMarker); idx >= 0 {
		return markerEnd + idx
	}

	end := markerEnd
	for end < len(src) && src[end] != '"' && src[end] != '\n' {
		end++
	}

	return end
}
