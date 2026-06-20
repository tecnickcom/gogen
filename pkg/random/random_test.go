package random

import (
	"errors"
	"io"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/require"
)

// shortReader returns fewer bytes than requested with a nil error on the first
// call, then reports EOF, simulating a custom io.Reader that performs partial
// reads. It is used to verify that RandomBytes does not silently truncate
// randomness when the underlying reader does not fully populate the buffer.
type shortReader struct {
	done bool
}

func (s *shortReader) Read(p []byte) (int, error) {
	if s.done || len(p) == 0 {
		return 0, io.EOF
	}

	s.done = true

	// Populate only the first byte, leaving the rest of p untouched.
	return 1, nil
}

func TestNew(t *testing.T) {
	t.Parallel()

	r := New(nil)

	require.NotNil(t, r.reader)
	require.NotNil(t, r.chrMap)

	errReader := iotest.ErrReader(errors.New("test-rand-reader-error"))
	re := New(
		errReader,
		WithByteToCharMap([]byte("0123456789abcdefx")),
	)

	require.NotNil(t, re.reader)
	require.Equal(t, errReader, re.reader)
	require.NotNil(t, re.chrMap)
	require.Len(t, re.chrMap, 17)
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

	// A reader that returns a short read with a nil error must be treated as an
	// error (io.ErrUnexpectedEOF) rather than silently truncating randomness.
	rs := New(&shortReader{})

	b, err = rs.RandomBytes(4)

	require.Error(t, err)
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
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

func TestRandHex64(t *testing.T) {
	t.Parallel()

	r := New(nil)

	h := r.RandHex64()

	require.Len(t, h, 16)
}

func TestRandString64(t *testing.T) {
	t.Parallel()

	r := New(nil)

	s := r.RandString64()

	require.NotEmpty(t, s)
}

func TestRandString(t *testing.T) {
	t.Parallel()

	r := New(nil)

	s, err := r.RandString(17)

	require.NoError(t, err)
	require.Len(t, s, 17)

	rc := New(nil, WithByteToCharMap([]byte(chrDigits+chrLowercase)))

	sc, err := rc.RandString(16)

	require.NoError(t, err)
	require.Len(t, sc, 16)

	re := New(iotest.ErrReader(errors.New("test-randstring-error")))

	s, err = re.RandString(32)

	require.Error(t, err)
	require.Empty(t, s)
}
