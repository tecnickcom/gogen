package random

import (
	"errors"
	"io"
	"strings"
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

// sequentialReader deterministically emits the byte sequence 0,1,...,255,0,...
// allowing exact verification of the rejection-sampling character selection.
// When max > 0, it reports EOF after producing max bytes.
type sequentialReader struct {
	next     byte
	produced int
	max      int
}

func (s *sequentialReader) Read(p []byte) (int, error) {
	for i := range p {
		if s.max > 0 && s.produced >= s.max {
			if i == 0 {
				return 0, io.EOF
			}

			return i, nil
		}

		p[i] = s.next
		s.next++
		s.produced++
	}

	return len(p), nil
}

func TestRandStringUniformSelection(t *testing.T) {
	t.Parallel()

	// Character map of length 10: limit = 250, bytes 250-255 are rejected.
	r := New(&sequentialReader{}, WithByteToCharMap([]byte(chrDigits)))

	// The first 250 sequential bytes (0-249) are all accepted and map each
	// character exactly 25 times: a biased mapping would skew the counts.
	s, err := r.RandString(250)

	require.NoError(t, err)
	require.Len(t, s, 250)

	for _, c := range chrDigits {
		require.Equal(t, 25, strings.Count(s, string(c)), "character %q", c)
	}

	// Requesting 256 characters consumes bytes 0-255: the 6 bytes >= 250 are
	// rejected and replaced by a refill read returning bytes 0-5, so exactly
	// the characters '0'-'5' gain one extra occurrence.
	r2 := New(&sequentialReader{}, WithByteToCharMap([]byte(chrDigits)))

	s2, err := r2.RandString(256)

	require.NoError(t, err)
	require.Len(t, s2, 256)

	for i, c := range chrDigits {
		want := 25
		if i < 6 {
			want = 26
		}

		require.Equal(t, want, strings.Count(s2, string(c)), "character %q", c)
	}
}

func TestRandStringRefillError(t *testing.T) {
	t.Parallel()

	// The reader is exhausted while replacing rejected bytes: bytes 250-255
	// are all rejected, so the refill reads hit EOF and RandString must fail
	// instead of looping or returning a short string.
	r := New(&sequentialReader{max: 256}, WithByteToCharMap([]byte(chrDigits)))

	s, err := r.RandString(252)

	require.Error(t, err)
	require.Empty(t, s)
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

func TestRandomBytes_NegativeLength(t *testing.T) {
	t.Parallel()

	r := New(nil)

	// A negative length must return ErrNegativeLength instead of panicking in make().
	b, err := r.RandomBytes(-1)

	require.ErrorIs(t, err, ErrNegativeLength)
	require.Nil(t, b)
}

func TestRandString_NegativeLength(t *testing.T) {
	t.Parallel()

	r := New(nil)

	// A negative length must return ErrNegativeLength instead of panicking in make().
	s, err := r.RandString(-1)

	require.ErrorIs(t, err, ErrNegativeLength)
	require.Empty(t, s)
}
