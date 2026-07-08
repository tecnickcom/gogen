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
