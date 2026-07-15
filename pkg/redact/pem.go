package redact

import (
	"bytes"
)

// PEM boundary markers for private-key blocks.
const (
	pemBeginPrefix = "-----BEGIN "
	pemEndPrefix   = "-----END "
)

// pemPrivateKeyLabels are the label fragments that terminate a secret PEM BEGIN
// line: the plain "PRIVATE KEY-----" (RSA/EC/OPENSSH/ENCRYPTED/... private keys)
// and the "PRIVATE KEY BLOCK-----" form used by armored OpenPGP secret keys
// (gpg --export-secret-keys --armor). Public material such as CERTIFICATE or
// PUBLIC KEY blocks is deliberately left visible.
var pemPrivateKeyLabels = [][]byte{ //nolint:gochecknoglobals
	[]byte("PRIVATE KEY-----"),
	[]byte("PRIVATE KEY BLOCK-----"),
}

// maxPEMBeginScan bounds the window searched for a private-key label after a
// BEGIN marker; the longest real BEGIN line ("-----BEGIN PGP PRIVATE KEY
// BLOCK-----") is under 40 bytes. Bounding it keeps a line packed with BEGIN
// markers from being rescanned to the far end for each one (quadratic).
const maxPEMBeginScan = 64

// pemBeginLineIsPrivateKey reports whether a BEGIN line (without its trailing
// '\r') is exactly a private-key marker: the FIRST occurrence of a label must
// terminate the line. A line whose first label is not at the end carries
// trailing content (e.g. a whole key blob with escaped "\n" sequences, or an
// END marker on the same logical line) and is left to the inline rule; a plain
// HasSuffix would wrongly match the "PRIVATE KEY-----" tail of an END marker.
func pemBeginLineIsPrivateKey(line []byte) bool {
	for _, label := range pemPrivateKeyLabels {
		if idx := bytes.Index(line, label); idx >= 0 && idx+len(label) == len(line) {
			return true
		}
	}

	return false
}

// inlinePEMPrivateKeyLabelEnd returns the index just past a private-key label
// found within maxPEMBeginScan bytes of i, and true, or (0, false) when none is
// present in that window.
func inlinePEMPrivateKeyLabelEnd(src []byte, i int) (int, bool) {
	window := src[i:min(len(src), i+maxPEMBeginScan)]

	best := -1

	for _, label := range pemPrivateKeyLabels {
		if idx := bytes.Index(window, label); idx >= 0 {
			if end := i + idx + len(label); best < 0 || end < best {
				best = end
			}
		}
	}

	if best < 0 {
		return 0, false
	}

	return best, true
}

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
	if !pemBeginLineIsPrivateKey(line) {
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
// embedded mid-line, most commonly a JSON string value carrying a whole PEM
// blob with escaped "\n" sequences under a non-sensitive key name.
func (re *Redactor) appendRedactedInlinePEMKeyAt(src []byte, i int, dst []byte) (int, []byte, bool) {
	if !hasPrefixAt(src, i, pemBeginPrefix) {
		return 0, dst, false
	}

	// The label must identify secret material within the bounded BEGIN window.
	markerEnd, ok := inlinePEMPrivateKeyLabelEnd(src, i)
	if !ok {
		return 0, dst, false
	}

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

// maxPEMBodyScan bounds the search for the "-----END" marker of an inline key
// body. A PEM private-key body (even a 4096-bit RSA key, with escaped "\n" line
// breaks) is a few KB; searching further for END would let a line packed with
// unterminated BEGIN markers rescan the whole tail for each one (quadratic).
// When END is not found within the window the body is redacted through the
// value's closing '"' or the line end instead (see inlinePEMBodyEnd): that scan
// cannot go quadratic, because a long non-whitespace body is redacted and
// consumed in one pass rather than re-scanned.
const maxPEMBodyScan = 1 << 14

// inlinePEMBodyEnd returns the end of the key material following an inline
// BEGIN marker: the start of the "-----END" marker when present within the
// bounded window, else the closing '"' (JSON-embedded blob) or the line end.
// The END search is bounded (quadratic guard); the '"'/line-end fallback is
// not: stopping it at the window would truncate a large key body mid-base64 and
// leave its tail visible. The fallback still cannot go quadratic: it consumes
// whatever it scans, so it runs at most once over any span.
func inlinePEMBodyEnd(src []byte, markerEnd int) int {
	limit := min(len(src), markerEnd+maxPEMBodyScan)

	if idx := bytes.Index(src[markerEnd:limit], pemEndMarker); idx >= 0 {
		return markerEnd + idx
	}

	end := markerEnd
	for end < len(src) && src[end] != '"' && src[end] != '\n' {
		end++
	}

	return end
}
