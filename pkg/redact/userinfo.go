package redact

// maxUserinfoScan bounds the forward scan for the '@' that terminates a URL's
// userinfo component. Real userinfo is short; the cap keeps pathological
// "://"-containing lines from being scanned at length repeatedly.
const maxUserinfoScan = 256

// appendRedactedURLPasswordAt handles the ':' at src[colon] when it begins a
// "://" scheme separator: if the URL carries userinfo credentials
// (scheme://user:password@host), it appends everything through the user's
// trailing ':' followed by the redaction marker, returning the index of the
// '@' so scheme, user and host stay visible:
//
//	postgres://app:hunter2@db.example.com  ->  postgres://app:***@db.example.com
//
// URLs without a password (ftp://anonymous@host) and without userinfo at all
// are left untouched. This catches bare "scheme://user:password@host" DSNs in
// prose and error messages. A scheme-LESS driver DSN (e.g. go-sql-driver/mysql
// "user:password@tcp(host)/db") is not matched bare — anchoring on a plain
// "user:password@" without a scheme would fire on ordinary "key:value@..."
// text — but it is still redacted when it appears as a sensitive field value
// (dsn=..., "database_url":"...") via the URL-encoded and JSON rules.
func (re *Redactor) appendRedactedURLPasswordAt(src []byte, colon int, dst []byte) (int, []byte, bool) {
	if colon+2 >= len(src) || src[colon+1] != '/' || src[colon+2] != '/' {
		return 0, dst, false
	}

	pwColon, at := findUserinfoPassword(src, colon+3)
	if at < 0 || pwColon < 0 {
		return 0, dst, false
	}

	dst = append(dst, src[colon:pwColon+1]...)
	dst = append(dst, re.marker...)

	return at, dst, true
}

// findUserinfoPassword scans the URL authority starting at uiStart for a
// userinfo component, returning the index of the ':' introducing the password
// and the index of the terminating '@'. Matching url.Parse semantics, the
// LAST '@' before a boundary delimits the userinfo, so a password containing
// an unencoded '@' ("user:p@ss@host") is hidden in full. Either index is -1
// when the URL carries no credentials before a boundary or the scan cap.
//
// Reaching the scan cap before a natural boundary is a hard no-match: such a
// span is not a real URL authority, and matching inside a capped window would
// make the result depend on how much earlier text a previous redaction pass
// shrank (convergence).
func findUserinfoPassword(src []byte, uiStart int) (int, int) {
	pwColon := -1
	lastAt := -1
	limit := min(len(src), uiStart+maxUserinfoScan)

	for j := uiStart; j < limit; j++ {
		c := src[j]

		if c == '@' {
			lastAt = j

			continue
		}

		if isUserinfoBoundary(c) {
			return userinfoResult(pwColon, lastAt)
		}

		if c == ':' && pwColon < 0 {
			pwColon = j
		}
	}

	if limit < len(src) {
		return -1, -1
	}

	// End of input is a natural boundary.
	return userinfoResult(pwColon, lastAt)
}

// userinfoResult validates the scanned indexes: a first ':' after the last
// '@' belongs to the host:port, not a password.
func userinfoResult(pwColon, lastAt int) (int, int) {
	if pwColon >= 0 && pwColon > lastAt {
		return -1, lastAt
	}

	return pwColon, lastAt
}

// isUserinfoBoundary reports whether c cannot appear before the '@' of a URL
// userinfo component, terminating the scan. '/' ends the authority, and
// whitespace/quotes/angle brackets end the URL itself in prose or structured
// text. Sub-delims like '&' and '=' are legal in passwords and are allowed.
func isUserinfoBoundary(c byte) bool {
	switch c {
	case '/', '?', '#', ' ', '\t', '\n', '\r', '"', '\'', '<', '>':
		return true
	default:
		return false
	}
}
