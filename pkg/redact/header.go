package redact

// sensitiveHeaderValueStart reports whether the line beginning at src[i] looks
// like an HTTP header whose name is sensitive (Authorization, Cookie,
// X-Api-Key, X-Auth-Token, ...), returning the index of the first byte of the
// header value (just past the colon and optional inline whitespace).
//
// The header name may only contain header-name characters, so JSON or
// URL-encoded lines containing a colon are rejected cheaply (usually on their
// first byte) and left to the dedicated redaction rules. Name sensitivity uses
// the same tokenized keyword check as JSON and URL-encoded keys, including the
// shared nonSensitiveKeys allowlist (CSP/HSTS/CORS and auth-challenge headers),
// so ordinary non-secret headers stay visible on every surface.
func (re *Redactor) sensitiveHeaderValueStart(src []byte, i int) (int, bool) {
	nameEnd := i
	for nameEnd < len(src) && isHeaderNameByte(src[nameEnd]) {
		nameEnd++
	}

	if nameEnd == i {
		return 0, false
	}

	// Allow optional whitespace between the header name and the colon.
	colon := skipInlineSpaces(src, nameEnd)
	if colon >= len(src) || src[colon] != ':' {
		return 0, false
	}

	if !re.isSensitiveKey(src[i:nameEnd]) {
		return 0, false
	}

	return skipInlineSpaces(src, colon+1), true
}

func isHeaderNameByte(c byte) bool {
	return isASCIIAlphaNum(c) || c == '-' || c == '_'
}

// skipHeaderLinePrefix skips leading indentation and an optional single trace
// decoration ("> " or "< ", as emitted per header line by `curl -v`, go-resty
// trace mode, and similar RoundTripper loggers) before a header name. The prefix
// is not consumed from the output (the caller emits src[i:valueStart] verbatim),
// so the "> " marker and indentation are preserved. A continuation/obs-fold line
// still has no "name:" shape, so it is not matched.
func skipHeaderLinePrefix(src []byte, i int) int {
	j := skipInlineSpaces(src, i)
	if j < len(src) && (src[j] == '>' || src[j] == '<') {
		j = skipInlineSpaces(src, j+1)
	}

	return j
}

func skipInlineSpaces(src []byte, i int) int {
	for i < len(src) && (src[i] == ' ' || src[i] == '\t') {
		i++
	}

	return i
}

// headerValueEnd returns the end of a redacted header value: the index of the
// line terminator, excluding a CRLF's '\r'. The caller resumes the main scan
// there, so the original terminator bytes ('\r\n' or '\n', as produced by
// httputil.DumpRequest/DumpResponse) are copied verbatim instead of being
// silently collapsed.
func headerValueEnd(src []byte, valueStart int) int {
	end := valueStart
	for end < len(src) && src[end] != '\n' {
		end++
	}

	if end < len(src) && end > valueStart && src[end-1] == '\r' {
		end--
	}

	return end
}
