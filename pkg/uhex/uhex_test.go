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

func TestHex32(t *testing.T) {
	t.Parallel()

	require.Equal(t, []byte("00000000"), Hex32(uint32(0)))
	require.Equal(t, []byte("000000ff"), Hex32(uint32(255)))
	require.Equal(t, []byte("01234567"), Hex32(uint32(0x01234567)))
	require.Equal(t, []byte("0f0f0f0f"), Hex32(uint32(0x0f0f0f0f)))
	require.Equal(t, []byte("ffffffff"), Hex32(uint32(0xffffffff)))
}

func TestHex16(t *testing.T) {
	t.Parallel()

	require.Equal(t, []byte("0000"), Hex16(uint16(0)))
	require.Equal(t, []byte("00ff"), Hex16(uint16(255)))
	require.Equal(t, []byte("0123"), Hex16(uint16(0x0123)))
	require.Equal(t, []byte("0f0f"), Hex16(uint16(0x0f0f)))
	require.Equal(t, []byte("ffff"), Hex16(uint16(0xffff)))
}

func TestHex8(t *testing.T) {
	t.Parallel()

	require.Equal(t, []byte("00"), Hex8(uint8(0)))
	require.Equal(t, []byte("ff"), Hex8(uint8(255)))
	require.Equal(t, []byte("01"), Hex8(uint8(0x01)))
	require.Equal(t, []byte("0f"), Hex8(uint8(0x0f)))
	require.Equal(t, []byte("ff"), Hex8(uint8(0xff)))
}
