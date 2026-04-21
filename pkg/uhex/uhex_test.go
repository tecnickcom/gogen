package uhex

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
