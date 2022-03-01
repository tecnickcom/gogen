package randkey

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	a := New()
	b := New()

	if a == b {
		t.Errorf("Two random keys should be different")
	}
}

func TestKey(t *testing.T) {
	t.Parallel()

	k := &RandKey{key: 255}

	require.Equal(t, uint64(0xff), k.Key())
}

func TestString(t *testing.T) {
	t.Parallel()

	k := &RandKey{key: 255}

	require.Equal(t, "73", k.String())
}

func TestHex(t *testing.T) {
	t.Parallel()

	k := &RandKey{key: 255}

	require.Equal(t, "00000000000000ff", k.Hex())

	k = &RandKey{key: uint64(0xffffffffffffffff)}

	require.Equal(t, "ffffffffffffffff", k.Hex())
}
