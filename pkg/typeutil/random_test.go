package typeutil

import (
	"errors"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/require"
)

//nolint:paralleltest
func TestRandomBytes(t *testing.T) {
	b, err := RandomBytes(RandReader, 32)

	require.NoError(t, err)
	require.Len(t, b, 32)

	rr := RandReader
	defer func() { RandReader = rr }()

	RandReader = iotest.ErrReader(errors.New("test-rand-reader-error"))

	b, err = RandomBytes(RandReader, 4)

	require.Error(t, err)
	require.Nil(t, b)
}

//nolint:paralleltest
func TestRandUint32(t *testing.T) {
	u := RandUint32()

	require.NotZero(t, u)

	rr := RandReader
	defer func() { RandReader = rr }()

	RandReader = iotest.ErrReader(errors.New("test-randuint32-error"))

	u = RandUint32()

	require.NotZero(t, u)
}

//nolint:paralleltest
func TestRandUint64(t *testing.T) {
	u := RandUint64()

	require.NotZero(t, u)

	rr := RandReader
	defer func() { RandReader = rr }()

	RandReader = iotest.ErrReader(errors.New("test-randuint64-error"))

	u = RandUint64()

	require.NotZero(t, u)
}
