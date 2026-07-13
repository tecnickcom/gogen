package redact

const (
	// jwtPrefix is the base64url encoding of `{"` — every JWS/JWE compact
	// token starts with it, making detection precise and cheap.
	jwtPrefix = "eyJ"

	// minJWTHeaderLen is the minimum plausible length of the base64url-encoded
	// JOSE header segment (a minimal real header like {"alg":"none"} encodes
	// to 20 bytes; 12 leaves margin while rejecting short prose).
	minJWTHeaderLen = 12

	// minJWTSegmentLen is the minimum length of the second (payload) segment.
	minJWTSegmentLen = 3
)

// appendRedactedJWTAt handles an 'e' at src[i]: when it starts a JWT compact
// serialization at a word boundary ("eyJ<header>.<payload>[.<segment>...]"),
// the whole token is replaced with the marker and the index just past it is
// returned. JWTs leak constantly through query parameters, state values, and
// error messages; the shape gates keep prose ("eyJ only") and glued
// identifiers untouched. All dot-joined trailing segments are consumed so JWE
// tokens (5 segments) do not leave ciphertext fragments behind.
func (re *Redactor) appendRedactedJWTAt(src []byte, i int, dst []byte) (int, []byte, bool) {
	if i > 0 && isWordChar(src[i-1]) {
		return 0, dst, false
	}

	end := jwtTokenEnd(src, i)
	if end == 0 {
		return 0, dst, false
	}

	dst = append(dst, re.marker...)

	return end, dst, true
}

// jwtTokenEnd returns the index just past a JWT compact serialization starting
// at src[i], or 0 when the required "eyJ<header>.<payload>" shape is absent.
// All further dot-joined segments are consumed so JWE tokens do not leave
// ciphertext fragments behind.
//
//nolint:gocyclo,cyclop // Sequential segment-shape checks; splitting would obscure the format.
func jwtTokenEnd(src []byte, i int) int {
	if !hasPrefixAt(src, i, jwtPrefix) {
		return 0
	}

	seg1End := scanBase64URL(src, i)
	if seg1End-i < minJWTHeaderLen || seg1End >= len(src) || src[seg1End] != '.' {
		return 0
	}

	seg2End := scanBase64URL(src, seg1End+1)
	seg2Len := seg2End - (seg1End + 1)

	// The classic two-segment form (header.payload, including unsigned alg=none)
	// needs a real payload. An EMPTY middle segment is valid only when at least
	// one further '.'-joined segment follows: this is the "dir" JWE compact form
	// (header..iv.ciphertext.tag) and detached-payload JWS (header..signature),
	// which would otherwise leak whole.
	hasNextSegment := seg2End < len(src) && src[seg2End] == '.' &&
		seg2End+1 < len(src) && isBase64URLByte(src[seg2End+1])

	seg2OK := seg2Len >= minJWTSegmentLen || (seg2Len == 0 && hasNextSegment)
	if !seg2OK {
		return 0
	}

	end := seg2End
	for end < len(src) && src[end] == '.' && end+1 < len(src) && isBase64URLByte(src[end+1]) {
		end = scanBase64URL(src, end+1)
	}

	return end
}

// scanBase64URL returns the index just past the run of base64url bytes
// starting at i.
func scanBase64URL(src []byte, i int) int {
	for i < len(src) && isBase64URLByte(src[i]) {
		i++
	}

	return i
}

func isBase64URLByte(c byte) bool {
	return isASCIIAlphaNum(c) || c == '-' || c == '_'
}
