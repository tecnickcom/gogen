package random

import (
	"testing"

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
