package jwt

import (
	"encoding/base64"
	"strconv"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestSigningMethod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		method   SigningMethod
		wantAlg  string
		wantSize int
	}{
		{SigningMethodHS256, "HS256", 32},
		{SigningMethodHS384, "HS384", 48},
		{SigningMethodHS512, "HS512", 64},
		{SigningMethod(99), "", 0},
	}

	for _, tt := range tests {
		require.Equal(t, tt.wantAlg, tt.method.Alg())
		require.Equal(t, tt.wantSize, tt.method.hashSize())

		if tt.wantSize == 0 {
			require.Nil(t, tt.method.hashNew())
		} else {
			require.NotNil(t, tt.method.hashNew())
			require.Equal(t, tt.wantSize, tt.method.hashNew()().Size())
		}
	}
}

func TestParseTokenErrors(t *testing.T) {
	t.Parallel()

	c, err := New(testKey, testVerify)
	require.NotNil(t, c)
	require.NoError(t, err)

	b64 := func(s string) string {
		return base64.RawURLEncoding.EncodeToString([]byte(s))
	}

	future := strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10)
	past := strconv.FormatInt(time.Now().Add(-time.Minute).Unix(), 10)

	tests := []struct {
		name    string
		token   string
		wantErr error
	}{
		{
			name:    "no separator",
			token:   "abc",
			wantErr: ErrMalformedToken,
		},
		{
			name:    "one separator",
			token:   "a.b",
			wantErr: ErrMalformedToken,
		},
		{
			name:    "too many segments",
			token:   "a.b.c.d",
			wantErr: ErrMalformedToken,
		},
		{
			name:    "invalid header encoding",
			token:   "!!!." + b64("{}") + ".sig",
			wantErr: ErrMalformedToken,
		},
		{
			name:    "invalid header JSON",
			token:   b64("{") + "." + b64("{}") + ".sig",
			wantErr: ErrMalformedToken,
		},
		{
			name:    "unexpected algorithm",
			token:   b64(`{"alg":"HS384","typ":"JWT"}`) + "." + b64("{}") + ".sig",
			wantErr: ErrUnexpectedSigningMethod,
		},
		{
			name:    "invalid signature encoding",
			token:   c.encodedHeader + "." + b64("{}") + ".!!!",
			wantErr: ErrMalformedToken,
		},
		{
			name:    "invalid signature",
			token:   c.encodedHeader + "." + b64("{}") + "." + b64("bogus-signature"),
			wantErr: ErrInvalidSignature,
		},
		{
			name:    "invalid payload encoding",
			token:   craftToken(t, c, "!!!"),
			wantErr: ErrMalformedToken,
		},
		{
			name:    "invalid claims payload",
			token:   craftToken(t, c, b64("junk")),
			wantErr: ErrMalformedToken,
		},
		{
			name:    "invalid numeric date in claims",
			token:   craftToken(t, c, b64(`{"exp":"nope"}`)),
			wantErr: ErrMalformedToken,
		},
		{
			name:    "missing expiration",
			token:   craftToken(t, c, b64(`{}`)),
			wantErr: ErrMissingExpiration,
		},
		{
			name:    "null expiration",
			token:   craftToken(t, c, b64(`{"exp":null}`)),
			wantErr: ErrMissingExpiration,
		},
		{
			name:    "out-of-range expiration",
			token:   craftToken(t, c, b64(`{"exp":1e300}`)),
			wantErr: ErrMalformedToken,
		},
		{
			name:    "expired token",
			token:   craftToken(t, c, b64(`{"exp":`+past+`}`)),
			wantErr: ErrTokenExpired,
		},
		{
			// RFC 7519 §4.1.4: exp is exclusive, so a token expiring at the
			// current second boundary is already expired at validation time.
			name:    "expired at current instant",
			token:   craftToken(t, c, b64(`{"exp":`+strconv.FormatInt(time.Now().Unix(), 10)+`}`)),
			wantErr: ErrTokenExpired,
		},
		{
			name:    "not yet valid token",
			token:   craftToken(t, c, b64(`{"exp":`+future+`,"nbf":`+future+`}`)),
			wantErr: ErrTokenNotYetValid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			claims, perr := c.parseToken(tt.token)
			require.NotNil(t, claims)
			require.ErrorIs(t, perr, tt.wantErr)
		})
	}
}

func TestParseTokenAcceptsInteropForms(t *testing.T) {
	t.Parallel()

	// The parser must accept RFC-permitted encodings that this package does not
	// emit itself: fractional NumericDates and a bare-string audience.
	c, err := New(testKey, testVerify, WithClaimAudience([]string{"aud-1"}))
	require.NotNil(t, c)
	require.NoError(t, err)

	b64 := func(s string) string {
		return base64.RawURLEncoding.EncodeToString([]byte(s))
	}

	future := strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10)

	claims, err := c.parseToken(craftToken(t, c, b64(`{"exp":`+future+`.5,"aud":"aud-1"}`)))
	require.NoError(t, err)
	require.Equal(t, Audience{"aud-1"}, claims.Audience)
	require.Equal(t, 500*time.Millisecond, time.Duration(claims.ExpiresAt.Nanosecond()))
}

func TestParseTokenNullClaims(t *testing.T) {
	t.Parallel()

	c, err := New(testKey, testVerify)
	require.NotNil(t, c)
	require.NoError(t, err)

	b64 := func(s string) string {
		return base64.RawURLEncoding.EncodeToString([]byte(s))
	}

	future := strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10)

	// JSON null on the optional claims is treated as absent (the pointer/slice
	// stays nil), so a token with only a valid exp is accepted.
	claims, err := c.parseToken(craftToken(t, c, b64(`{"exp":`+future+`,"aud":null,"nbf":null}`)))
	require.NoError(t, err)
	require.Nil(t, claims.Audience)
	require.Nil(t, claims.NotBefore)
}

func TestIssueToken(t *testing.T) {
	t.Parallel()

	c, err := New(testKey, testVerify,
		WithClaimIssuer("iss-1"),
		WithClaimAudience([]string{"aud-1"}),
	)
	require.NotNil(t, c)
	require.NoError(t, err)

	token, err := c.IssueToken("cli-user")
	require.NoError(t, err)

	claims, err := c.parseToken(token)
	require.NoError(t, err)

	require.Equal(t, "cli-user", claims.Username)
	require.Equal(t, "cli-user", claims.Subject)
	require.Equal(t, "iss-1", claims.Issuer)
	require.Equal(t, Audience{"aud-1"}, claims.Audience)
	require.NotNil(t, claims.ExpiresAt)
	require.NotNil(t, claims.AuthTime)
	require.NotEmpty(t, claims.ID)
}

func TestVerifyToken(t *testing.T) {
	t.Parallel()

	c, err := New(testKey, testVerify)
	require.NotNil(t, c)
	require.NoError(t, err)

	token, err := c.IssueToken("raw-user")
	require.NoError(t, err)

	// A valid raw token string yields its verified claims.
	claims, err := c.VerifyToken(token)
	require.NoError(t, err)
	require.Equal(t, "raw-user", claims.Username)
	require.Equal(t, "raw-user", claims.Subject)

	// A tampered token is rejected (the padded signature still decodes, but no
	// longer matches).
	got, err := c.VerifyToken(token + "x")
	require.NotNil(t, got)
	require.ErrorIs(t, err, ErrInvalidSignature)

	got, err = c.VerifyToken("")
	require.NotNil(t, got)
	require.ErrorIs(t, err, ErrMalformedToken)
}

func TestSignTokenErrors(t *testing.T) {
	t.Parallel()

	c, err := New(testKey, testVerify)
	require.NotNil(t, c)
	require.NoError(t, err)

	// A claim date that cannot be encoded surfaces as a marshaling error.
	_, err = c.signToken(&Claims{ExpiresAt: NewNumericDate(time.Unix(-10, 0))})
	require.ErrorIs(t, err, ErrInvalidNumericDate)

	// White-box: an invalid signing method surfaces from the signature
	// computation, and makes signature verification fail closed.
	c.signingMethod = SigningMethod(99)

	_, err = c.signToken(&Claims{ExpiresAt: NewNumericDate(time.Now().Add(time.Minute))})
	require.ErrorIs(t, err, ErrInvalidSigningMethod)

	require.False(t, c.verifySignature("input", []byte("sig")))
}

func TestTokenRoundTrip(t *testing.T) {
	t.Parallel()

	for _, method := range []SigningMethod{SigningMethodHS256, SigningMethodHS384, SigningMethodHS512} {
		key := make([]byte, method.hashSize())

		c, err := New(key, testVerify,
			WithSigningMethod(method),
			WithClaimIssuer("iss-1"),
			WithClaimAudience([]string{"aud-1"}),
		)
		require.NotNil(t, c)
		require.NoError(t, err)

		token, err := c.signToken(c.newClaims("round-trip-user", nil))
		require.NoError(t, err)

		claims, err := c.parseToken(token)
		require.NoError(t, err)

		require.Equal(t, "round-trip-user", claims.Username)
		require.Equal(t, "round-trip-user", claims.Subject)
		require.Equal(t, "iss-1", claims.Issuer)
		require.Equal(t, Audience{"aud-1"}, claims.Audience)
		require.NotNil(t, claims.ExpiresAt)
		require.NotNil(t, claims.IssuedAt)
		require.NotNil(t, claims.NotBefore)
		require.NotNil(t, claims.AuthTime)
		require.NotEmpty(t, claims.ID)
	}
}

func TestCrossValidationWithReferenceLibrary(t *testing.T) {
	t.Parallel()

	c, err := New(testKey, testVerify,
		WithClaimIssuer("iss-1"),
		WithClaimAudience([]string{"aud-1", "aud-2"}),
	)
	require.NotNil(t, c)
	require.NoError(t, err)

	// Outbound interop: a token minted by this package must verify and decode
	// with the reference golang-jwt implementation.
	token, err := c.signToken(c.newClaims("interop-user", nil))
	require.NoError(t, err)

	refClaims := jwtv5.MapClaims{}
	_, err = jwtv5.ParseWithClaims(
		token,
		refClaims,
		func(_ *jwtv5.Token) (any, error) { return testKey, nil },
		jwtv5.WithValidMethods([]string{"HS256"}),
		jwtv5.WithExpirationRequired(),
		jwtv5.WithIssuer("iss-1"),
		jwtv5.WithAllAudiences("aud-1", "aud-2"),
	)
	require.NoError(t, err, "reference library must accept our token")
	require.Equal(t, "interop-user", refClaims["username"])
	require.Equal(t, "interop-user", refClaims["sub"])
	require.Contains(t, refClaims, "auth_time")
	require.Contains(t, refClaims, "jti")

	// Inbound interop: a token minted by the reference library must verify and
	// decode with this package, with every claim populated.
	now := time.Now()
	refToken := signClaimsV5(t, testKey, jwtv5.MapClaims{
		"iss":       "iss-1",
		"sub":       "interop-user",
		"aud":       []string{"aud-1", "aud-2"},
		"exp":       now.Add(time.Minute).Unix(),
		"nbf":       now.Add(-time.Minute).Unix(),
		"iat":       now.Add(-time.Minute).Unix(),
		"jti":       "test-jti",
		"username":  "interop-user",
		"auth_time": now.Add(-time.Minute).Unix(),
	})

	claims, err := c.parseToken(refToken)
	require.NoError(t, err, "our parser must accept the reference library token")
	require.Equal(t, "iss-1", claims.Issuer)
	require.Equal(t, "interop-user", claims.Subject)
	require.Equal(t, Audience{"aud-1", "aud-2"}, claims.Audience)
	require.Equal(t, "test-jti", claims.ID)
	require.Equal(t, "interop-user", claims.Username)
	require.Equal(t, now.Add(time.Minute).Unix(), claims.ExpiresAt.Unix())
	require.Equal(t, now.Add(-time.Minute).Unix(), claims.NotBefore.Unix())
	require.Equal(t, now.Add(-time.Minute).Unix(), claims.IssuedAt.Unix())
	require.Equal(t, now.Add(-time.Minute).Unix(), claims.AuthTime.Unix())
}

// craftToken signs an arbitrary payload segment with the instance signing key,
// producing a structurally attacker-shaped token that still carries a valid
// signature, so post-signature parse branches can be exercised.
func craftToken(t *testing.T, c *JWT, payloadSeg string) string {
	t.Helper()

	signingInput := c.encodedHeader + "." + payloadSeg

	sig, err := c.computeSignature(signingInput, c.key)
	require.NoError(t, err)

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}
