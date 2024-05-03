package random

import (
	"errors"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	r := New(nil)

	require.NotNil(t, r.reader)
}

func TestRandomBytes(t *testing.T) {
	t.Parallel()

	r := New(nil)

	b, err := r.RandomBytes(32)

	require.NoError(t, err)
	require.Len(t, b, 32)

	re := New(iotest.ErrReader(errors.New("test-rand-reader-error")))

	b, err = re.RandomBytes(4)

	require.Error(t, err)
	require.Nil(t, b)
}

func TestRandUint32(t *testing.T) {
	t.Parallel()

	r := New(nil)

	u := r.RandUint32()

	require.NotZero(t, u)

	re := New(iotest.ErrReader(errors.New("test-randuint32-error")))

	u = re.RandUint32()

	require.NotZero(t, u)
}

func TestRandUint64(t *testing.T) {
	t.Parallel()

	r := New(nil)

	u := r.RandUint64()

	require.NotZero(t, u)

	re := New(iotest.ErrReader(errors.New("test-randuint64-error")))

	u = re.RandUint64()

	require.NotZero(t, u)
}
