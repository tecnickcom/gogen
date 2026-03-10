package random

import (
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

func TestRnd_UID128_String(t *testing.T) {
	t.Parallel()

	r := New(nil)

	a := r.UID128().String()
	b := r.UID128().String()

	require.NotEqual(t, a, b)
}

func TestRnd_UID128_Hex_Collision(t *testing.T) {
	t.Parallel()

	r := New(nil)

	fn := func() string {
		return r.UID128().Hex()
	}

	collisionTest(t, fn, 10, 100)
}
