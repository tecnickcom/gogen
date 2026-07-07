package random

import (
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRnd_UID128(t *testing.T) {
	t.Parallel()

	r := New(nil)

	a := r.UID128()
	b := r.UID128()

	require.NotEqual(t, a, b)
}

func TestRnd_UID128_Hex(t *testing.T) {
	t.Parallel()

	r := New(nil)

	a := r.UID128().Hex()
	b := r.UID128().Hex()

	require.NotEqual(t, a, b)
	require.Len(t, a, 32)
	require.Len(t, b, 32)
}

func TestRnd_UID128_Format(t *testing.T) {
	t.Parallel()

	r := New(nil)
	u := r.UID128()

	// A pre-filled buffer must be fully overwritten.
	b := [32]byte{'x'}
	u.Format(&b)

	require.Equal(t, u.Hex(), string(b[:]))
	require.Len(t, b, 32)
}

func TestRnd_UID128_Byte(t *testing.T) {
	t.Parallel()

	r := New(nil)
	u := r.UID128()

	require.Equal(t, u.Hex(), string(u.Byte()))
	require.Len(t, u.Byte(), 32)
}

func TestRnd_UID128_String(t *testing.T) {
	t.Parallel()

	r := New(nil)

	a := r.UID128().String()
	b := r.UID128().String()

	require.NotEqual(t, a, b)
}

func TestRnd_UID128_String_MatchesFormatUint(t *testing.T) {
	t.Parallel()

	// String must stay byte-identical to FormatUint(t)+FormatUint(r); the
	// max/max case also exercises the full 26-byte buffer (13 base-36 digits each).
	cases := []TUID128{
		{t: 0, r: 0},
		{t: 1, r: 2},
		{t: 1234567890, r: 9876543210},
		{t: math.MaxUint64, r: math.MaxUint64},
	}

	for _, u := range cases {
		want := strconv.FormatUint(u.t, 36) + strconv.FormatUint(u.r, 36)
		require.Equal(t, want, u.String())
	}
}

func TestRnd_UID128_Hex_Collision(t *testing.T) {
	t.Parallel()

	r := New(nil)

	fn := func() string {
		return r.UID128().Hex()
	}

	collisionTest(t, fn, 10, 100)
}
