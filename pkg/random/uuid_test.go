package random

import (
	"bytes"
	"errors"
	"testing"
	"testing/iotest"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUUIDv7(t *testing.T) {
	t.Parallel()

	r := New(nil)

	a := r.UUIDv7()
	b := r.UUIDv7()

	require.NotEqual(t, a, b)
	require.Len(t, a, 16)
	require.Len(t, b, 16)

	// Version and variant must hold on the default (crypto/rand) path, not only on
	// the reader-failure path exercised by TestUUIDv7_ReaderError.
	for range 1000 {
		u := r.UUIDv7()

		require.Equal(t, byte(0x70), u[6]&0xF0, "version nibble must be 7")
		require.Equal(t, byte(0x80), u[8]&0xC0, "variant bits must be 0b10")
	}
}

// TestUUIDv7_Golden pins the exact byte layout against a hand-computed vector.
// Without this, the octet order of Format is only ever compared against itself
// (via String, which is built on Format) and any permutation would pass.
func TestUUIDv7_Golden(t *testing.T) {
	t.Parallel()

	u := UUID{
		0x01, 0x8f, 0x2e, 0x3b, 0x4c, 0x5d, 0x7a, 0xbc,
		0x8d, 0xef, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc,
	}

	const want = "018f2e3b-4c5d-7abc-8def-123456789abc"

	var dst [36]byte

	u.Format(&dst)

	require.Equal(t, want, string(dst[:]), "Format")
	require.Equal(t, want, string(u.Byte()), "Byte")
	require.Equal(t, want, u.String(), "String")
}

// TestUUIDv7_Timestamp asserts that unix_ts_ms really is the current time, encoded
// big-endian in octets 0-5. A reversed, frozen, or seconds-scaled timestamp would
// pass every other test in this file.
func TestUUIDv7_Timestamp(t *testing.T) {
	t.Parallel()

	r := New(nil)

	before := time.Now().UnixMilli()
	u := r.UUIDv7()
	after := time.Now().UnixMilli()

	ms := int64(u[0])<<40 | int64(u[1])<<32 | int64(u[2])<<24 |
		int64(u[3])<<16 | int64(u[4])<<8 | int64(u[5])

	require.GreaterOrEqual(t, ms, before, "timestamp must not predate the call")
	require.LessOrEqual(t, ms, after, "timestamp must not postdate the call")
}

// TestUUIDv7_Sortable asserts the headline property: consecutive values are
// non-decreasing. The comparison covers octets 0-7 (unix_ts_ms, version, rand_a)
// but not rand_b, which is random by design. This pins the timestamp byte order
// and the rand_a clock derivation together: zeroing rand_a, reversing the
// timestamp, or freezing the clock all break it.
func TestUUIDv7_Sortable(t *testing.T) {
	t.Parallel()

	r := New(nil)

	prev := r.UUIDv7()

	for i := range 20000 {
		cur := r.UUIDv7()

		require.LessOrEqual(t, bytes.Compare(prev[:8], cur[:8]), 0,
			"value %d went backwards: %s then %s", i, prev.String(), cur.String())

		prev = cur
	}
}

// TestUUIDv7_SubMillisecondPrecision asserts that rand_a carries increased clock
// precision (RFC 9562 6.2, Method 3) rather than being a constant: across a run it
// must visit many of its 4,096 buckets. Together with TestUUIDv7_Sortable, which
// requires it to be non-decreasing within a millisecond, this pins rand_a as a
// clock counter: zeroing it fails this test, and filling it with randomness fails
// the other.
func TestUUIDv7_SubMillisecondPrecision(t *testing.T) {
	t.Parallel()

	r := New(nil)

	seen := make(map[uint16]struct{})

	for range 20000 {
		u := r.UUIDv7()

		randA := uint16(u[6]&0x0F)<<8 | uint16(u[7])

		require.Less(t, randA, uint16(4096), "rand_a must fit in 12 bits")

		seen[randA] = struct{}{}
	}

	// A real sub-millisecond counter advances one bucket every ~244ns, so a run this
	// long visits hundreds of them. A constant visits exactly one.
	require.Greater(t, len(seen), 100,
		"rand_a took only %d distinct values: it is not carrying sub-millisecond clock precision", len(seen))
}

// TestUUIDv7_UsesConfiguredReader asserts that rand_b is taken from the configured
// reader. Without it, UUIDv7 could ignore the reader entirely and every other test
// would still pass.
func TestUUIDv7_UsesConfiguredReader(t *testing.T) {
	t.Parallel()

	entropy := []byte{0xAA, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}

	u := New(bytes.NewReader(entropy)).UUIDv7()

	// Octet 8 keeps the low 6 bits of the first entropy byte under the variant.
	require.Equal(t, byte(0x80)|(0x3F&entropy[0]), u[8], "rand_b octet 8")
	require.Equal(t, entropy[1:8], u[9:16], "rand_b octets 9-15")
}

func TestUUIDv7_ReaderError(t *testing.T) {
	t.Parallel()

	// A failing random reader must not panic; UUIDv7 falls back to RandUint64.
	r := New(iotest.ErrReader(errors.New("test-uuidv7-reader-error")))

	u := r.UUIDv7()

	require.Len(t, u, 16)
	require.Equal(t, byte(0x70), u[6]&0xF0, "version nibble must be 7")
	require.Equal(t, byte(0x80), u[8]&0xC0, "variant bits must be 0b10")
}

func TestUUIDv7_Byte(t *testing.T) {
	t.Parallel()

	r := New(nil)

	a := r.UUIDv7().Byte()
	b := r.UUIDv7().Byte()

	require.NotEqual(t, a, b)
	require.Len(t, a, 36)
	require.Len(t, b, 36)
}

func TestUUIDv7_Format_Empty(t *testing.T) {
	t.Parallel()

	r := New(nil)
	u := r.UUIDv7()

	var b [36]byte

	u.Format(&b)

	require.Equal(t, u.String(), string(b[:]))
	require.Len(t, b, 36)
}

func TestUUIDv7_Format_FillsBuffer(t *testing.T) {
	t.Parallel()

	r := New(nil)
	u := r.UUIDv7()

	var b [36]byte

	u.Format(&b)

	require.Equal(t, u.String(), string(b[:]))
}

func TestUUIDv7_Format_OverwriteBuffer(t *testing.T) {
	t.Parallel()

	r := New(nil)
	u := r.UUIDv7()
	b := [36]byte{'x'}

	u.Format(&b)

	require.Equal(t, u.String(), string(b[:]))
}

func TestUUIDv7_String(t *testing.T) {
	t.Parallel()

	r := New(nil)

	a := r.UUIDv7().String()
	b := r.UUIDv7().String()

	require.NotEqual(t, a, b)
	require.Len(t, a, 36)
	require.Len(t, b, 36)
}

func TestUUIDv7_Collision(t *testing.T) {
	t.Parallel()

	r := New(nil)

	f := func() string {
		return r.UUIDv7().String()
	}

	collisionTest(t, f, 100, 1000)
}
