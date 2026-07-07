package uhex

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// sweep64 returns a broad set of uint64 test patterns: edge values plus a
// pseudo-random sweep that exercises every byte position.
func sweep64() []uint64 {
	seed := []uint64{0, 1, 255, 0x0123456789abcdef, 0xfedcba9876543210, 0xffffffffffffffff}

	vals := make([]uint64, 0, len(seed)+2*4096)
	vals = append(vals, seed...)

	for i := range uint64(4096) {
		vals = append(vals, i*0x9e3779b97f4a7c15^(i<<29), 1<<(i%64))
	}

	return vals
}

func TestHex64(t *testing.T) {
	t.Parallel()

	require.Equal(t, []byte("0000000000000000"), Hex64(uint64(0)))
	require.Equal(t, []byte("00000000000000ff"), Hex64(uint64(255)))
	require.Equal(t, []byte("0123456789abcdef"), Hex64(uint64(0x0123456789abcdef)))
	require.Equal(t, []byte("0f0f0f0f0f0f0f0f"), Hex64(uint64(0x0f0f0f0f0f0f0f0f)))
	require.Equal(t, []byte("ffffffffffffffff"), Hex64(uint64(0xffffffffffffffff)))
}

func TestHex64UB(t *testing.T) {
	t.Parallel()

	dst := [16]byte{}
	Hex64UB(uint64(0x0123456789abcdef), &dst)
	require.Equal(t, []byte("0123456789abcdef"), dst[:])
}

func TestHex64B(t *testing.T) {
	t.Parallel()

	require.Equal(t, []byte("0000000000000000"), Hex64B([8]byte{}))
	require.Equal(t, []byte("00000000000000ff"), Hex64B([8]byte{0, 0, 0, 0, 0, 0, 0, 0xff}))
	require.Equal(t, []byte("0123456789abcdef"), Hex64B([8]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}))
	require.Equal(t, []byte("0f0f0f0f0f0f0f0f"), Hex64B([8]byte{0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f}))
	require.Equal(t, []byte("ffffffffffffffff"), Hex64B([8]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}))
}

func TestHex64BB(t *testing.T) {
	t.Parallel()

	dst := [16]byte{}
	Hex64BB([8]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}, &dst)
	require.Equal(t, []byte("0123456789abcdef"), dst[:])
}

func TestHex32(t *testing.T) {
	t.Parallel()

	require.Equal(t, []byte("00000000"), Hex32(uint32(0)))
	require.Equal(t, []byte("000000ff"), Hex32(uint32(255)))
	require.Equal(t, []byte("01234567"), Hex32(uint32(0x01234567)))
	require.Equal(t, []byte("0f0f0f0f"), Hex32(uint32(0x0f0f0f0f)))
	require.Equal(t, []byte("ffffffff"), Hex32(uint32(0xffffffff)))
}

func TestHex32UB(t *testing.T) {
	t.Parallel()

	dst := [8]byte{}
	Hex32UB(uint32(0x89abcdef), &dst)
	require.Equal(t, []byte("89abcdef"), dst[:])
}

func TestHex32B(t *testing.T) {
	t.Parallel()

	require.Equal(t, []byte("00000000"), Hex32B([4]byte{}))
	require.Equal(t, []byte("000000ff"), Hex32B([4]byte{0, 0, 0, 0xff}))
	require.Equal(t, []byte("01234567"), Hex32B([4]byte{0x01, 0x23, 0x45, 0x67}))
	require.Equal(t, []byte("0f0f0f0f"), Hex32B([4]byte{0x0f, 0x0f, 0x0f, 0x0f}))
	require.Equal(t, []byte("ffffffff"), Hex32B([4]byte{0xff, 0xff, 0xff, 0xff}))
}

func TestHex32BB(t *testing.T) {
	t.Parallel()

	dst := [8]byte{}
	Hex32BB([4]byte{0x89, 0xab, 0xcd, 0xef}, &dst)
	require.Equal(t, []byte("89abcdef"), dst[:])
}

func TestHex16(t *testing.T) {
	t.Parallel()

	require.Equal(t, []byte("0000"), Hex16(uint16(0)))
	require.Equal(t, []byte("00ff"), Hex16(uint16(255)))
	require.Equal(t, []byte("0123"), Hex16(uint16(0x0123)))
	require.Equal(t, []byte("0f0f"), Hex16(uint16(0x0f0f)))
	require.Equal(t, []byte("ffff"), Hex16(uint16(0xffff)))
}

func TestHex16UB(t *testing.T) {
	t.Parallel()

	dst := [4]byte{}
	Hex16UB(uint16(0x0f0f), &dst)
	require.Equal(t, []byte("0f0f"), dst[:])
}

func TestHex16B(t *testing.T) {
	t.Parallel()

	require.Equal(t, []byte("0000"), Hex16B([2]byte{}))
	require.Equal(t, []byte("00ff"), Hex16B([2]byte{0, 0xff}))
	require.Equal(t, []byte("0123"), Hex16B([2]byte{0x01, 0x23}))
	require.Equal(t, []byte("0f0f"), Hex16B([2]byte{0x0f, 0x0f}))
	require.Equal(t, []byte("ffff"), Hex16B([2]byte{0xff, 0xff}))
}

func TestHex16BB(t *testing.T) {
	t.Parallel()

	dst := [4]byte{}
	Hex16BB([2]byte{0x0f, 0x0f}, &dst)
	require.Equal(t, []byte("0f0f"), dst[:])
}

func TestHex8(t *testing.T) {
	t.Parallel()

	require.Equal(t, []byte("00"), Hex8(uint8(0)))
	require.Equal(t, []byte("ff"), Hex8(uint8(255)))
	require.Equal(t, []byte("01"), Hex8(uint8(0x01)))
	require.Equal(t, []byte("0f"), Hex8(uint8(0x0f)))
	require.Equal(t, []byte("ff"), Hex8(uint8(0xff)))
}

func TestHex8UB(t *testing.T) {
	t.Parallel()

	dst := [2]byte{}
	Hex8UB(uint8(0xff), &dst)
	require.Equal(t, []byte("ff"), dst[:])
}

func TestHex8B(t *testing.T) {
	t.Parallel()

	require.Equal(t, []byte("00"), Hex8B([1]byte{}))
	require.Equal(t, []byte("ff"), Hex8B([1]byte{0xff}))
	require.Equal(t, []byte("01"), Hex8B([1]byte{0x01}))
	require.Equal(t, []byte("0f"), Hex8B([1]byte{0x0f}))
}

func TestHex8BB(t *testing.T) {
	t.Parallel()

	dst := [2]byte{}
	Hex8BB([1]byte{0xff}, &dst)
	require.Equal(t, []byte("ff"), dst[:])
}

// TestHex64Oracle cross-checks the 64-bit integer encoders against fmt.Sprintf.
func TestHex64Oracle(t *testing.T) {
	t.Parallel()

	for _, n := range sweep64() {
		want := []byte(fmt.Sprintf("%016x", n))

		require.Equal(t, want, Hex64(n))

		var dst [16]byte

		Hex64UB(n, &dst)
		require.Equal(t, want, dst[:])
	}
}

// TestHex32Oracle cross-checks the 32-bit integer encoders against fmt.Sprintf.
func TestHex32Oracle(t *testing.T) {
	t.Parallel()

	for _, n := range sweep64() {
		v := uint32(n)
		want := []byte(fmt.Sprintf("%08x", v))

		require.Equal(t, want, Hex32(v))

		var dst [8]byte

		Hex32UB(v, &dst)
		require.Equal(t, want, dst[:])
	}
}

// TestHex16Oracle exhaustively cross-checks the 16-bit integer encoders.
func TestHex16Oracle(t *testing.T) {
	t.Parallel()

	for n := range 0x10000 {
		v := uint16(n)
		want := []byte(fmt.Sprintf("%04x", v))

		require.Equal(t, want, Hex16(v))

		var dst [4]byte

		Hex16UB(v, &dst)
		require.Equal(t, want, dst[:])
	}
}

// TestHex8Oracle exhaustively cross-checks the 8-bit integer encoders.
func TestHex8Oracle(t *testing.T) {
	t.Parallel()

	for n := range 0x100 {
		v := uint8(n)
		want := []byte(fmt.Sprintf("%02x", v))

		require.Equal(t, want, Hex8(v))

		var dst [2]byte

		Hex8UB(v, &dst)
		require.Equal(t, want, dst[:])
	}
}

// TestHexBytesOracle sweeps every byte value across positions and cross-checks
// the byte-array encoders against encoding/hex.
func TestHexBytesOracle(t *testing.T) {
	t.Parallel()

	for i := range 256 {
		b8 := [8]byte{byte(i), byte(i * 7), byte(i + 3), byte(255 - i), byte(i ^ 0x5a), byte(i << 1), byte(i >> 1), byte(i * 31)}
		want8 := []byte(hex.EncodeToString(b8[:]))
		require.Equal(t, want8, Hex64B(b8))

		var d16 [16]byte

		Hex64BB(b8, &d16)
		require.Equal(t, want8, d16[:])

		b4 := [4]byte{byte(i), byte(255 - i), byte(i ^ 0x33), byte(i * 5)}
		want4 := []byte(hex.EncodeToString(b4[:]))
		require.Equal(t, want4, Hex32B(b4))

		var d8 [8]byte

		Hex32BB(b4, &d8)
		require.Equal(t, want4, d8[:])

		b2 := [2]byte{byte(i), byte(255 - i)}
		want2 := []byte(hex.EncodeToString(b2[:]))
		require.Equal(t, want2, Hex16B(b2))

		var d4 [4]byte

		Hex16BB(b2, &d4)
		require.Equal(t, want2, d4[:])

		b1 := [1]byte{byte(i)}
		want1 := []byte(hex.EncodeToString(b1[:]))
		require.Equal(t, want1, Hex8B(b1))

		var d2 [2]byte

		Hex8BB(b1, &d2)
		require.Equal(t, want1, d2[:])
	}
}
