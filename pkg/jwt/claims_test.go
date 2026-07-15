package jwt

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNumericDateMarshalJSON(t *testing.T) {
	t.Parallel()

	// A regular date encodes as integer seconds, truncating sub-second precision.
	d := NewNumericDate(time.Unix(1700000000, 999999999))

	b, err := d.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, "1700000000", string(b))

	// Pre-epoch dates (including the zero time) are rejected.
	_, err = NewNumericDate(time.Unix(-10, 0)).MarshalJSON()
	require.ErrorIs(t, err, ErrInvalidNumericDate)

	_, err = NewNumericDate(time.Time{}).MarshalJSON()
	require.ErrorIs(t, err, ErrInvalidNumericDate)

	// The 2^53 boundary is exactly representable and accepted; one second past
	// it is rejected, keeping the encode side symmetric with the decode side so
	// every encodable date is also decodable.
	b, err = NewNumericDate(time.Unix(maxNumericDateSec, 0)).MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, "9007199254740992", string(b))

	_, err = NewNumericDate(time.Unix(maxNumericDateSec+1, 0)).MarshalJSON()
	require.ErrorIs(t, err, ErrInvalidNumericDate)
}

func TestNumericDateUnmarshalJSON(t *testing.T) {
	t.Parallel()

	var d NumericDate

	// Integer seconds.
	require.NoError(t, d.UnmarshalJSON([]byte("1700000000")))
	require.Equal(t, int64(1700000000), d.Unix())

	// Fractional seconds are preserved (RFC 7519 permits non-integer values).
	require.NoError(t, d.UnmarshalJSON([]byte("1700000000.5")))
	require.Equal(t, int64(1700000000), d.Unix())
	require.InDelta(t, 500*time.Millisecond, d.Nanosecond(), float64(time.Millisecond))

	// Non-numeric values are rejected.
	require.ErrorIs(t, d.UnmarshalJSON([]byte(`"nope"`)), ErrInvalidNumericDate)

	// Non-finite numbers (ParseFloat range error) are rejected.
	require.ErrorIs(t, d.UnmarshalJSON([]byte("1e999")), ErrInvalidNumericDate)

	// Finite but out-of-representable-range numbers are rejected, so the
	// seconds-to-int64 conversion stays well-defined across architectures.
	require.ErrorIs(t, d.UnmarshalJSON([]byte("1e300")), ErrInvalidNumericDate)
	require.ErrorIs(t, d.UnmarshalJSON([]byte("-1e300")), ErrInvalidNumericDate)

	// A large but legitimate far-future timestamp (year ~3000) still parses.
	require.NoError(t, d.UnmarshalJSON([]byte("32503680000")))
	require.Equal(t, int64(32503680000), d.Unix())

	// The exact 2^53 boundary is accepted on decode too.
	require.NoError(t, d.UnmarshalJSON([]byte("9007199254740992")))
	require.Equal(t, maxNumericDateSec, d.Unix())
}

func TestAudienceUnmarshalJSON(t *testing.T) {
	t.Parallel()

	var a Audience

	// RFC 7519 §4.1.3: a single string is a valid audience encoding.
	require.NoError(t, a.UnmarshalJSON([]byte(`"solo"`)))
	require.Equal(t, Audience{"solo"}, a)

	// The array form.
	require.NoError(t, a.UnmarshalJSON([]byte(`["a","b"]`)))
	require.Equal(t, Audience{"a", "b"}, a)

	// Invalid inputs on both branches.
	require.Error(t, a.UnmarshalJSON([]byte(`"unterminated`)))
	require.Error(t, a.UnmarshalJSON([]byte(`123`)))
}

func TestAudienceMarshalsAsArray(t *testing.T) {
	t.Parallel()

	// A single-value audience marshals as an array, matching the golang-jwt v5
	// default wire format this package is interoperable with.
	b, err := json.Marshal(Audience{"one"})
	require.NoError(t, err)
	require.JSONEq(t, `["one"]`, string(b))
}

func TestNewClaimsClampsExpToSessionLifetime(t *testing.T) {
	t.Parallel()

	const expiration = 10 * time.Minute

	tests := []struct {
		name        string
		maxLifetime time.Duration
		authAge     time.Duration // session age; 0 means a fresh login
		clamped     bool
	}{
		{"unset cap keeps full expiration", 0, 0, false},
		{"cap wider than expiration does not fire on login", 30 * time.Minute, 0, false},
		{"cap narrower than expiration shortens login", 4 * time.Minute, 0, true},
		{"renew near the cap is clamped to the cap", 30 * time.Minute, 29 * time.Minute, true},
		{"renew early in the session keeps full expiration", 30 * time.Minute, 1 * time.Minute, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := New(testKey, testVerify,
				WithExpirationTime(expiration),
				WithMaxSessionLifetime(tt.maxLifetime),
			)
			require.NoError(t, err)

			var authTime *NumericDate
			if tt.authAge > 0 {
				authTime = NewNumericDate(time.Now().UTC().Add(-tt.authAge))
			}

			before := time.Now().UTC()
			claims := c.newClaims("clamp-user", authTime)
			after := time.Now().UTC()

			require.NotNil(t, claims.ExpiresAt)

			if !tt.clamped {
				// exp tracks now + expirationTime, unclamped.
				require.WithinRange(t, claims.ExpiresAt.Time, before.Add(expiration), after.Add(expiration))

				return
			}

			// Clamped: exp never outlives the cap measured from the session start
			// (the preserved auth_time on renewal, or the issue time on a login).
			sessionStart := after
			if authTime != nil {
				sessionStart = authTime.Time
			}

			require.LessOrEqual(t, claims.ExpiresAt.Unix(), sessionStart.Add(tt.maxLifetime).Unix(),
				"exp must not outlive session start + maxSessionLifetime")
		})
	}
}

func TestNewClaimsClampedTokenStillValidates(t *testing.T) {
	t.Parallel()

	// A token minted right at the clamp boundary (renewal granted just under the
	// cap) must still be strictly in the future, so it validates immediately.
	c, err := New(testKey, testVerify,
		WithExpirationTime(10*time.Minute),
		WithMaxSessionLifetime(5*time.Minute),
	)
	require.NoError(t, err)

	// Session started 5 minutes minus one second ago: the clamp yields exp about
	// one second in the future.
	authTime := NewNumericDate(time.Now().UTC().Add(-5*time.Minute + time.Second))

	token, err := c.signToken(c.newClaims("boundary-user", authTime))
	require.NoError(t, err)

	claims, err := c.parseToken(token)
	require.NoError(t, err, "a token clamped to the cap must still validate right after issue")
	require.True(t, claims.ExpiresAt.After(time.Now()))
}
