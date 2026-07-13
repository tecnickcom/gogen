package random

import (
	"bytes"
	"encoding/binary"
	"math"
	"strconv"
	"testing"
	"time"

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

// TestRnd_UID128_Golden pins the exact hexadecimal layout against a hand-computed
// vector: the time half must come first, most-significant nibble first, which is
// what makes the hexadecimal form time-ordered. Swapping the halves would pass
// every assertion that compares Format against Hex, since Hex is built on Format.
func TestRnd_UID128_Golden(t *testing.T) {
	t.Parallel()

	u := TUID128{t: 0x0123456789abcdef, r: 0xfedcba9876543210}

	const want = "0123456789abcdef" + "fedcba9876543210"

	var dst [32]byte

	u.Format(&dst)

	require.Equal(t, want, string(dst[:]), "Format")
	require.Equal(t, want, string(u.Byte()), "Byte")
	require.Equal(t, want, u.Hex(), "Hex")
}

// TestRnd_UID128_Layout asserts the documented split: the high 64 bits are the
// generation time in Unix nanoseconds, and the low 64 bits come from the reader.
func TestRnd_UID128_Layout(t *testing.T) {
	t.Parallel()

	entropy := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	before := time.Now().UnixNano()
	u := New(bytes.NewReader(entropy)).UID128()
	after := time.Now().UnixNano()

	require.GreaterOrEqual(t, u.t, uint64(before), "time half must not predate the call")
	require.LessOrEqual(t, u.t, uint64(after), "time half must not postdate the call")

	require.Equal(t, binary.LittleEndian.Uint64(entropy), u.r,
		"random half must come from the configured reader")
}

// TestRnd_UID128_StringAmbiguity pins the ambiguity the String doc warns about:
// two distinct values can render as the same base-36 string, which is why Hex is
// the round-trippable form.
func TestRnd_UID128_StringAmbiguity(t *testing.T) {
	t.Parallel()

	a := TUID128{t: 1, r: 75} // "1" + "23"
	b := TUID128{t: 38, r: 3} // "12" + "3"

	require.NotEqual(t, a, b)
	require.Equal(t, a.String(), b.String(), "distinct values collide in String, as documented")
	require.NotEqual(t, a.Hex(), b.Hex(), "Hex must stay unambiguous")
}

func TestRnd_UID128_Hex_Collision(t *testing.T) {
	t.Parallel()

	r := New(nil)

	fn := func() string {
		return r.UID128().Hex()
	}

	collisionTest(t, fn, 10, 100)
}
