package jwt

// This file contains the RFC 7515 compact JWS wire format: the HMAC signing
// methods, token serialization/signing, and parsing/verification/validation.

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"slices"
	"strings"
	"time"
)

// Supported HMAC signing methods.
const (
	// SigningMethodHS256 is HMAC using SHA-256 (the default).
	SigningMethodHS256 SigningMethod = iota

	// SigningMethodHS384 is HMAC using SHA-384.
	SigningMethodHS384

	// SigningMethodHS512 is HMAC using SHA-512.
	SigningMethodHS512
)

// Token parsing and validation errors returned by [JWT.Authenticate].
var (
	// ErrMalformedToken is returned when a token is not a syntactically valid
	// RFC 7515 compact JWS (wrong segment count, bad base64url, invalid JSON).
	ErrMalformedToken = errors.New("jwt: malformed token")

	// ErrTokenTooLarge is returned when a token exceeds the configured maximum
	// size ([WithMaxTokenBytes]), both when verifying an oversize token and when
	// signing would mint one (e.g. from an unusually long username). It lets a
	// caller distinguish a size overrun from a genuinely malformed token.
	ErrTokenTooLarge = errors.New("jwt: token exceeds the maximum size")

	// ErrUnexpectedSigningMethod is returned when the token header declares an
	// algorithm different from the configured one (e.g. alg=none or an
	// algorithm-confusion attempt).
	ErrUnexpectedSigningMethod = errors.New("jwt: unexpected signing method")

	// ErrUnsupportedCritHeader is returned when the JOSE header carries a `crit`
	// parameter (RFC 7515 §4.1.11). This package understands no critical
	// extensions, so any `crit` member, including the invalid empty array, is fatal.
	ErrUnsupportedCritHeader = errors.New("jwt: unsupported critical header parameter")

	// ErrDuplicateHeaderParameter is returned when the JOSE header repeats a
	// member name. RFC 7515 §4 requires Header Parameter names to be unique and
	// permits a verifier to reject duplicates rather than silently take the last.
	ErrDuplicateHeaderParameter = errors.New("jwt: duplicate JOSE header parameter")

	// ErrInvalidSignature is returned when the token signature does not verify
	// under any of the accepted keys.
	ErrInvalidSignature = errors.New("jwt: invalid token signature")

	// ErrMissingExpiration is returned when a parsed token lacks an expiration time.
	ErrMissingExpiration = errors.New("jwt: token missing expiration time")

	// ErrTokenExpired is returned when the token expiration time has passed.
	ErrTokenExpired = errors.New("jwt: token expired")

	// ErrTokenNotYetValid is returned when the token not-before time is in the future.
	ErrTokenNotYetValid = errors.New("jwt: token not yet valid")

	// ErrInvalidIssuer is returned when the token issuer does not match the configured one.
	ErrInvalidIssuer = errors.New("jwt: invalid token issuer")

	// ErrInvalidAudience is returned when the token audience does not include
	// every configured audience.
	ErrInvalidAudience = errors.New("jwt: invalid token audience")
)

// SigningMethod selects the HMAC-SHA2 algorithm used to sign and verify tokens.
//
// Only symmetric HMAC methods (RFC 7518 §3.2) are supported: the same secret
// key both signs and verifies. The zero value is [SigningMethodHS256].
type SigningMethod uint8

// Alg returns the RFC 7518 `alg` header parameter value for the method
// ("HS256", "HS384" or "HS512"), or an empty string when the method is not one
// of the supported constants.
func (m SigningMethod) Alg() string {
	switch m {
	case SigningMethodHS256:
		return "HS256"
	case SigningMethodHS384:
		return "HS384"
	case SigningMethodHS512:
		return "HS512"
	default:
		return ""
	}
}

// hashNew returns the constructor of the hash backing the method, or nil when
// the method is not one of the supported constants.
func (m SigningMethod) hashNew() func() hash.Hash {
	switch m {
	case SigningMethodHS256:
		return sha256.New
	case SigningMethodHS384:
		return sha512.New384
	case SigningMethodHS512:
		return sha512.New
	default:
		return nil
	}
}

// hashSize returns the hash output size in bytes, which is also the minimum
// signing key length per RFC 7518 §3.2, or 0 when the method is invalid.
func (m SigningMethod) hashSize() int {
	switch m {
	case SigningMethodHS256:
		return sha256.Size
	case SigningMethodHS384:
		return sha512.Size384
	case SigningMethodHS512:
		return sha512.Size
	default:
		return 0
	}
}

// IssueToken signs and returns a fresh token for username, using the same claim
// recipe as the login flow (exp, iat, nbf, jti, sub, iss, aud, auth_time).
//
// It is the issuance counterpart to [JWT.Authenticate]: use it to mint a token
// outside the HTTP login flow (test fixtures, CLI tools, service-to-service
// calls, WebSocket handshakes) once the caller has verified the user's identity
// by its own means. It does not verify credentials.
func (c *JWT) IssueToken(username string) (string, error) {
	return c.signToken(c.newClaims(username, nil))
}

// VerifyToken parses, verifies, and validates a raw compact-JWS token string,
// returning its claims.
//
// It is the transport-independent counterpart to [JWT.Authenticate]: use it for
// tokens that arrive outside an HTTP Authorization header (WebSocket messages,
// queue payloads, gRPC metadata). The returned error is for logging or
// branching; do not echo it to clients (it may reveal token-validation
// internals). The returned claims are never nil; they are populated only when
// the token signature verified.
func (c *JWT) VerifyToken(tokenString string) (*Claims, error) {
	return c.parseToken(tokenString)
}

// initTokenConfig precomputes the encoded JOSE header for the configured
// signing method and the ordered verification key list (the current signing
// key first, then any previous keys still accepted for rotation).
func (c *JWT) initTokenConfig() {
	// Clone the key material so later mutation (or zeroization) of the caller's
	// buffers cannot silently change what this instance signs or verifies.
	c.key = bytes.Clone(c.key)
	for i, key := range c.previousKeys {
		c.previousKeys[i] = bytes.Clone(key)
	}

	c.encodedHeader = base64.RawURLEncoding.EncodeToString(
		[]byte(`{"alg":"` + c.signingMethod.Alg() + `","typ":"JWT"}`),
	)
	c.verifyKeys = append([][]byte{c.key}, c.previousKeys...)
}

// signToken serializes claims and signs them as an RFC 7515 compact JWS
// (base64url(header) "." base64url(claims) "." base64url(signature)).
func (c *JWT) signToken(claims *Claims) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("jwt: unable to encode claims: %w", err)
	}

	signingInput := c.encodedHeader + "." + base64.RawURLEncoding.EncodeToString(payload)

	sig, err := c.computeSignature(signingInput, c.key)
	if err != nil {
		return "", err
	}

	token := signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)

	// Never mint a token the verify path would reject as oversize: the login and
	// issuance inputs (e.g. the username) are not individually bounded, so a long
	// one could otherwise produce a token larger than maxTokenBytes.
	if len(token) > c.maxTokenBytes {
		return "", fmt.Errorf("%w: %d bytes exceeds the %d-byte limit", ErrTokenTooLarge, len(token), c.maxTokenBytes)
	}

	return token, nil
}

// computeSignature returns the HMAC of signingInput under key using the
// configured signing method.
func (c *JWT) computeSignature(signingInput string, key []byte) ([]byte, error) {
	hashNew := c.signingMethod.hashNew()
	if hashNew == nil {
		return nil, ErrInvalidSigningMethod
	}

	mac := hmac.New(hashNew, key)
	_, _ = mac.Write([]byte(signingInput)) // hash.Hash.Write never returns an error

	return mac.Sum(nil), nil
}

// parseToken parses a compact JWS, verifies its signature against the accepted
// keys, and validates the claims. The signature is verified BEFORE the claims
// payload is decoded, so attacker-controlled JSON never reaches the decoder.
// The returned claims are never nil; they are populated only when the token
// signature verified.
func (c *JWT) parseToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	// Bound the input before any decoding: the JOSE header must be base64- and
	// JSON-decoded before the signature can be checked, so an unbounded token
	// would let an unauthenticated caller force a large decode.
	if len(tokenString) > c.maxTokenBytes {
		return claims, fmt.Errorf("%w: %d bytes exceeds the %d-byte limit", ErrTokenTooLarge, len(tokenString), c.maxTokenBytes)
	}

	headerSeg, rest, found := strings.Cut(tokenString, ".")
	if !found {
		return claims, ErrMalformedToken
	}

	payloadSeg, sigSeg, found := strings.Cut(rest, ".")
	if !found || strings.Contains(sigSeg, ".") {
		return claims, ErrMalformedToken
	}

	err := c.checkHeader(headerSeg)
	if err != nil {
		return claims, err
	}

	sig, err := base64.RawURLEncoding.Strict().DecodeString(sigSeg)
	if err != nil {
		return claims, fmt.Errorf("%w: invalid signature encoding: %w", ErrMalformedToken, err)
	}

	signingInput := tokenString[:len(headerSeg)+1+len(payloadSeg)]
	if !c.verifySignature(signingInput, sig) {
		return claims, ErrInvalidSignature
	}

	payload, err := base64.RawURLEncoding.Strict().DecodeString(payloadSeg)
	if err != nil {
		return claims, fmt.Errorf("%w: invalid payload encoding: %w", ErrMalformedToken, err)
	}

	err = json.Unmarshal(payload, claims)
	if err != nil {
		return claims, fmt.Errorf("%w: invalid claims payload: %w", ErrMalformedToken, err)
	}

	return claims, c.validateClaims(claims)
}

// joseHeader holds the JOSE header parameters this package inspects: the signing
// algorithm and the presence of a critical-extensions parameter.
type joseHeader struct {
	alg  string
	crit json.RawMessage
}

// checkHeader decodes the JOSE header segment and enforces the configured
// signing algorithm, rejecting alg=none and algorithm-confusion tokens. A `crit`
// parameter (RFC 7515 §4.1.11) or a duplicate member name is rejected; all other
// parameters (typ, kid, ...) are ignored, since with a pinned symmetric
// algorithm and mandatory signature verification they cannot alter how a token
// is processed.
func (c *JWT) checkHeader(headerSeg string) error {
	headerJSON, err := base64.RawURLEncoding.Strict().DecodeString(headerSeg)
	if err != nil {
		return fmt.Errorf("%w: invalid header encoding: %w", ErrMalformedToken, err)
	}

	header, err := decodeJOSEHeader(headerJSON)
	if err != nil {
		return err
	}

	// RFC 7515 §4.1.11: a JWS carrying critical extensions the recipient does not
	// understand MUST be rejected. This package understands none, so any crit
	// member, including "crit": null and the invalid empty array, is fatal.
	if len(header.crit) > 0 {
		return ErrUnsupportedCritHeader
	}

	if header.alg != c.signingMethod.Alg() {
		return fmt.Errorf("%w: %q", ErrUnexpectedSigningMethod, header.alg)
	}

	return nil
}

// decodeJOSEHeader parses the JOSE header JSON in a single strict pass. It
// extracts alg and crit and rejects a header that is not a JSON object, repeats
// a member name, or carries trailing data. RFC 7515 §4 requires unique Header
// Parameter names and lets a verifier reject duplicates rather than silently
// take the last value, which encoding/json would.
func decodeJOSEHeader(headerJSON []byte) (joseHeader, error) {
	var header joseHeader

	dec := json.NewDecoder(bytes.NewReader(headerJSON))

	tok, err := dec.Token()
	if err != nil {
		return header, fmt.Errorf("%w: invalid header: %w", ErrMalformedToken, err)
	}

	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return header, fmt.Errorf("%w: header is not a JSON object", ErrMalformedToken)
	}

	seen := make(map[string]struct{})

	for dec.More() {
		err = header.readMember(dec, seen)
		if err != nil {
			return header, err
		}
	}

	// Consume the closing delimiter and confirm nothing follows it, so a
	// truncated or padded header segment is rejected rather than half-accepted.
	_, err = dec.Token()
	if err != nil {
		return header, fmt.Errorf("%w: invalid header: %w", ErrMalformedToken, err)
	}

	_, err = dec.Token()
	if !errors.Is(err, io.EOF) {
		return header, fmt.Errorf("%w: trailing data after header", ErrMalformedToken)
	}

	return header, nil
}

// readMember reads one "name": value pair into header, rejecting a member name
// already present in seen. Values are consumed whole (including nested objects),
// so a nested member name is never mistaken for a header parameter.
func (h *joseHeader) readMember(dec *json.Decoder, seen map[string]struct{}) error {
	tok, err := dec.Token()

	// A member name in a JSON object is always a string: Token returns an error
	// (handled here) rather than a non-string token at a member-name position.
	name, ok := tok.(string)
	if err != nil || !ok {
		return fmt.Errorf("%w: invalid header member name", ErrMalformedToken)
	}

	if _, dup := seen[name]; dup {
		return fmt.Errorf("%w: %q", ErrDuplicateHeaderParameter, name)
	}

	seen[name] = struct{}{}

	var value json.RawMessage

	err = dec.Decode(&value)
	if err != nil {
		return fmt.Errorf("%w: invalid header: %w", ErrMalformedToken, err)
	}

	switch name {
	case "alg":
		err = json.Unmarshal(value, &h.alg)
		if err != nil {
			return fmt.Errorf("%w: invalid alg: %w", ErrMalformedToken, err)
		}
	case "crit":
		h.crit = value
	}

	return nil
}

// verifySignature reports whether sig is a valid signature of signingInput
// under any of the accepted verification keys (the current signing key plus any
// configured previous keys). Each comparison is constant-time.
func (c *JWT) verifySignature(signingInput string, sig []byte) bool {
	for _, key := range c.verifyKeys {
		expected, err := c.computeSignature(signingInput, key)
		if err != nil {
			return false
		}

		if hmac.Equal(sig, expected) {
			return true
		}
	}

	return false
}

// validateClaims enforces the time claims (exp is mandatory, nbf honored when
// present) with the configured clock-skew leeway, and the issuer and audience
// claims when configured.
func (c *JWT) validateClaims(claims *Claims) error {
	now := time.Now()

	if claims.ExpiresAt == nil {
		return ErrMissingExpiration
	}

	// RFC 7519 §4.1.4: the current time MUST be strictly before exp (rejecting
	// at the exact instant), matching the golang-jwt reference implementation.
	if !now.Before(claims.ExpiresAt.Add(c.clockSkewLeeway)) {
		return ErrTokenExpired
	}

	if claims.NotBefore != nil && now.Add(c.clockSkewLeeway).Before(claims.NotBefore.Time) {
		return ErrTokenNotYetValid
	}

	if c.issuer != "" && claims.Issuer != c.issuer {
		return ErrInvalidIssuer
	}

	for _, aud := range c.audience {
		if !slices.Contains(claims.Audience, aud) {
			return ErrInvalidAudience
		}
	}

	return nil
}

// expiredOnIssue reports whether exp, once truncated to whole seconds as it will
// be at signing ([NumericDate.MarshalJSON]), already fails the verification
// expiry check (RFC 7519 §4.1.4 with the configured leeway). It lets issuance
// avoid minting a born-expired token when the session-cap clamp lands within the
// current second.
func (c *JWT) expiredOnIssue(exp *NumericDate) bool {
	if exp == nil {
		return false
	}

	flooredExp := time.Unix(exp.Unix(), 0)

	return !time.Now().Before(flooredExp.Add(c.clockSkewLeeway))
}
