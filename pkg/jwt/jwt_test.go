package jwt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

// testKey is a 32-byte HMAC key, satisfying the HS256 minimum key length.
var testKey = []byte("0123456789abcdef0123456789abcdef") //nolint:gochecknoglobals

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      []byte
		verifyFn VerifyCredentialsFn
		opts     []Option
		wantErr  error
	}{
		{
			name:     "success with default options",
			key:      testKey,
			verifyFn: testVerify,
		},
		{
			name:     "success with custom options",
			key:      testKey,
			verifyFn: testVerify,
			opts: []Option{
				WithExpirationTime(1 * time.Minute),
				WithRenewTime(10 * time.Second),
				WithMaxBodyBytes(4096),
				WithSendResponseFn(func(_ context.Context, _ http.ResponseWriter, _ int, _ string) {}),
			},
		},
		{
			name:     "failure with empty key",
			verifyFn: testVerify,
			wantErr:  ErrEmptyKey,
		},
		{
			name:    "failure with nil verify function",
			key:     testKey,
			wantErr: ErrNilVerifyFn,
		},
		{
			name:     "failure with invalid signing method",
			key:      testKey,
			verifyFn: testVerify,
			opts:     []Option{WithSigningMethod(SigningMethod(99))},
			wantErr:  ErrInvalidSigningMethod,
		},
		{
			name:     "failure with weak key",
			key:      []byte("too-short"),
			verifyFn: testVerify,
			wantErr:  ErrWeakKey,
		},
		{
			name:     "failure with weak previous key",
			key:      testKey,
			verifyFn: testVerify,
			opts:     []Option{WithPreviousKeys([]byte("too-short"))},
			wantErr:  ErrWeakKey,
		},
		{
			name:     "success with valid previous key",
			key:      testKey,
			verifyFn: testVerify,
			opts:     []Option{WithPreviousKeys([]byte("fedcba9876543210fedcba9876543210"))},
		},
		{
			name:     "failure with non-positive expiration",
			key:      testKey,
			verifyFn: testVerify,
			opts:     []Option{WithExpirationTime(0)},
			wantErr:  ErrInvalidExpirationTime,
		},
		{
			name:     "failure with sub-second expiration",
			key:      testKey,
			verifyFn: testVerify,
			opts:     []Option{WithExpirationTime(500 * time.Millisecond)},
			wantErr:  ErrShortExpirationTime,
		},
		{
			name:     "success with one-second expiration",
			key:      testKey,
			verifyFn: testVerify,
			opts:     []Option{WithExpirationTime(time.Second), WithRenewTime(500 * time.Millisecond)},
		},
		{
			name:     "failure with non-positive renew",
			key:      testKey,
			verifyFn: testVerify,
			opts:     []Option{WithRenewTime(-1)},
			wantErr:  ErrInvalidRenewTime,
		},
		{
			name:     "failure with non-positive max body bytes",
			key:      testKey,
			verifyFn: testVerify,
			opts:     []Option{WithMaxBodyBytes(0)},
			wantErr:  ErrInvalidMaxBodyBytes,
		},
		{
			name:     "failure with non-positive max token bytes",
			key:      testKey,
			verifyFn: testVerify,
			opts:     []Option{WithMaxTokenBytes(0)},
			wantErr:  ErrInvalidMaxTokenBytes,
		},
		{
			name:     "failure with negative max session lifetime",
			key:      testKey,
			verifyFn: testVerify,
			opts:     []Option{WithMaxSessionLifetime(-1)},
			wantErr:  ErrInvalidMaxSessionLifetime,
		},
		{
			name:     "failure with sub-second max session lifetime",
			key:      testKey,
			verifyFn: testVerify,
			opts:     []Option{WithMaxSessionLifetime(500 * time.Millisecond)},
			wantErr:  ErrShortMaxSessionLifetime,
		},
		{
			name:     "success with one-second max session lifetime",
			key:      testKey,
			verifyFn: testVerify,
			opts:     []Option{WithMaxSessionLifetime(time.Second)},
		},
		{
			name:     "failure with negative clock skew leeway",
			key:      testKey,
			verifyFn: testVerify,
			opts:     []Option{WithClockSkewLeeway(-1)},
			wantErr:  ErrInvalidClockSkewLeeway,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := New(tt.key, tt.verifyFn, tt.opts...)
			if tt.wantErr != nil {
				require.Nil(t, c, "New() returned value should be nil")
				require.ErrorIs(t, err, tt.wantErr, "New() error = %v, want %v", err, tt.wantErr)

				return
			}

			require.NotNil(t, c, "New() returned value should not be nil")
			require.NoError(t, err, "New() unexpected error = %v", err)
		})
	}
}

func TestWeakKeyErrorNamesOffender(t *testing.T) {
	t.Parallel()

	// A short signing key names the signing key.
	_, err := New([]byte("too-short"), testVerify)
	require.ErrorIs(t, err, ErrWeakKey)
	require.Contains(t, err.Error(), "signing key")

	// A short previous key names it by rotation index, so an operator can tell
	// which buffer is short (here the second previous key, index 1).
	_, err = New(testKey, testVerify, WithPreviousKeys(testKey, []byte("short")))
	require.ErrorIs(t, err, ErrWeakKey)
	require.Contains(t, err.Error(), "previous key 1")
	require.NotContains(t, err.Error(), "signing key")
}

func TestNormalizeOptionals(t *testing.T) {
	t.Parallel()

	// Nil/empty optional values must restore defaults, not brick the handlers.
	c, err := New(
		testKey,
		testVerify,
		WithLogger(nil),
		WithSendResponseFn(nil),
		WithAuthorizationHeader(""),
	)
	require.NotNil(t, c)
	require.NoError(t, err)
	require.NotNil(t, c.logger)
	require.NotNil(t, c.sendResponseFn)
	require.Equal(t, DefaultAuthorizationHeader, c.authorizationHeader)

	rr := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "/", strings.NewReader(`{"username":"test-name", "password":"test-name"}`))

	require.NotPanics(t, func() { c.LoginHandler(rr, req) })

	resp := rr.Result()
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestNewCopiesKeyMaterial(t *testing.T) {
	t.Parallel()

	prevOriginal := []byte("old-key-89abcdef0123456789abcdef")

	// Hand New/WithPreviousKeys mutable buffers, as a caller reusing or
	// zeroizing its key material would.
	key := append([]byte(nil), testKey...)
	prev := append([]byte(nil), prevOriginal...)

	c, err := New(key, testVerify, WithPreviousKeys(prev))
	require.NotNil(t, c)
	require.NoError(t, err)

	token, err := c.IssueToken("test-name")
	require.NoError(t, err)

	oldToken := signClaimsV5(t, prevOriginal, jwtv5.MapClaims{
		"username": "test-name",
		"exp":      time.Now().Add(time.Minute).Unix(),
	})

	// Zeroize the caller's buffers: the instance must be unaffected.
	clear(key)
	clear(prev)

	_, err = c.VerifyToken(token)
	require.NoError(t, err, "tokens signed before the mutation must still verify")

	_, err = c.VerifyToken(oldToken)
	require.NoError(t, err, "previous-key tokens must still verify after the mutation")

	token2, err := c.IssueToken("test-name")
	require.NoError(t, err)

	_, err = c.VerifyToken(token2)
	require.NoError(t, err, "newly issued tokens must verify with the retained key")
}

// signClaimsV5 signs claims with the reference golang-jwt implementation (HMAC
// SHA-256), so every crafted-token test doubles as an interoperability check of
// this package's parser against the reference library.
func signClaimsV5(t *testing.T, key []byte, claims jwtv5.MapClaims) string {
	t.Helper()

	signed, err := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims).SignedString(key)
	require.NoError(t, err)

	return signed
}

// testVerify treats a login as valid when the password equals a non-empty username.
func testVerify(username, password string) (bool, error) {
	if username == "" {
		return false, nil
	}

	return password == username, nil
}
