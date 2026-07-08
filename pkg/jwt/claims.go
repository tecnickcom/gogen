package jwt

// This file contains the token payload data model: the Claims structure and
// the JSON encodings of the RFC 7519 claim value types.

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"
)

// maxNumericDateSec bounds accepted timestamps to 2^53 seconds: the float64
// integer-precision limit (beyond which seconds are not exactly representable)
// and about the year 285,428,751, far past any legitimate claim. It also keeps
// the seconds-to-int64 conversion below well-defined, avoiding the
// implementation-dependent result of converting an out-of-range float.
const maxNumericDateSec int64 = 1 << 53

// maxNumericDate is the same bound as a float64, for the decode-side check
// where the input arrives as a parsed JSON number.
const maxNumericDate = float64(maxNumericDateSec)

// ErrInvalidNumericDate is returned when a numeric date cannot be encoded or
// decoded (non-numeric JSON value, or a pre-epoch time on encoding).
var ErrInvalidNumericDate = errors.New("jwt: invalid numeric date")

// NumericDate is an RFC 7519 §2 NumericDate: a JSON number holding the seconds
// elapsed since the Unix epoch. It embeds [time.Time], so all Time methods are
// available on it.
type NumericDate struct {
	time.Time
}

// NewNumericDate wraps t as a [NumericDate].
func NewNumericDate(t time.Time) *NumericDate {
	return &NumericDate{Time: t}
}

// MarshalJSON encodes the date as integer seconds since the Unix epoch,
// truncating sub-second precision. Pre-epoch dates (including the zero time)
// are rejected with [ErrInvalidNumericDate]: registered time claims never
// legitimately predate 1970, so such a value indicates a bug (typically an
// uninitialized time). Dates past 2^53 seconds are rejected symmetrically with
// [NumericDate.UnmarshalJSON], so every encodable date is also decodable.
func (d *NumericDate) MarshalJSON() ([]byte, error) {
	sec := d.Unix()
	if sec < 0 || sec > maxNumericDateSec {
		return nil, ErrInvalidNumericDate
	}

	return strconv.AppendInt(nil, sec, 10), nil
}

// UnmarshalJSON decodes integer or fractional JSON numbers (RFC 7519 allows
// both), preserving sub-second precision when present. Out-of-range values are
// rejected with [ErrInvalidNumericDate].
func (d *NumericDate) UnmarshalJSON(data []byte) error {
	f, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidNumericDate, err)
	}

	if f < -maxNumericDate || f > maxNumericDate {
		return fmt.Errorf("%w: timestamp out of range", ErrInvalidNumericDate)
	}

	sec, frac := math.Modf(f)
	d.Time = time.Unix(int64(sec), int64(frac*float64(time.Second)))

	return nil
}

// Audience is the RFC 7519 §4.1.3 `aud` claim: one or more case-sensitive
// strings identifying the intended token recipients. It marshals as a JSON
// array and unmarshals from either a single JSON string or an array of strings,
// as the RFC permits both forms.
type Audience []string

// UnmarshalJSON decodes either a bare JSON string or an array of strings.
func (a *Audience) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		var single string

		err := json.Unmarshal(data, &single)
		if err != nil {
			return fmt.Errorf("jwt: invalid audience: %w", err)
		}

		*a = Audience{single}

		return nil
	}

	var values []string

	err := json.Unmarshal(data, &values)
	if err != nil {
		return fmt.Errorf("jwt: invalid audience: %w", err)
	}

	*a = values

	return nil
}

// Claims holds the JWT payload: the RFC 7519 registered claims plus the
// package-specific `username` and `auth_time` claims.
type Claims struct {
	// Issuer is the `iss` (Issuer) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.1
	Issuer string `json:"iss,omitempty"`

	// Subject is the `sub` (Subject) claim: the authenticated principal.
	// See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.2
	Subject string `json:"sub,omitempty"`

	// Audience is the `aud` (Audience) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.3
	Audience Audience `json:"aud,omitempty"`

	// ExpiresAt is the `exp` (Expiration Time) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.4
	ExpiresAt *NumericDate `json:"exp,omitempty"`

	// NotBefore is the `nbf` (Not Before) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.5
	NotBefore *NumericDate `json:"nbf,omitempty"`

	// IssuedAt is the `iat` (Issued At) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.6
	IssuedAt *NumericDate `json:"iat,omitempty"`

	// ID is the `jti` (JWT ID) claim. See https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.7
	ID string `json:"jti,omitempty"`

	// Username is the authenticated principal. It duplicates the `sub` claim,
	// which is authoritative; Username is retained for consumers that read it.
	Username string `json:"username"`

	// AuthTime records the time of the original login and is preserved across
	// renewals, so [WithMaxSessionLifetime] can bound the total session length.
	// It follows the OpenID Connect `auth_time` claim (OIDC Core §2).
	AuthTime *NumericDate `json:"auth_time,omitempty"`
}

// newClaims builds fresh token claims for username, with expiration, issue, and
// not-before times relative to the current time and a new unique token ID. The
// authTime is preserved across renewals; when nil (a fresh login) it defaults to
// the current time.
func (c *JWT) newClaims(username string, authTime *NumericDate) *Claims {
	tnow := time.Now().UTC()

	if authTime == nil {
		authTime = NewNumericDate(tnow)
	}

	return &Claims{
		ExpiresAt: NewNumericDate(tnow.Add(c.expirationTime)), // exp
		IssuedAt:  NewNumericDate(tnow),                       // iat
		NotBefore: NewNumericDate(tnow),                       // nbf
		ID:        c.rnd.UUIDv7().String(),                    // jti
		Issuer:    c.issuer,                                   // iss
		Subject:   username,                                   // sub: the authenticated principal
		Audience:  Audience(c.audience),                       // aud
		Username:  username,
		AuthTime:  authTime, // auth_time: original login, preserved across renewals
	}
}
