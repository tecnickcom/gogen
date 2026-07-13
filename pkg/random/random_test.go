package random

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"strings"
	"sync/atomic"
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

// dripReader delivers exactly one byte per call with a nil error, forever: a
// legal io.Reader that only ever performs short reads. Reads must be retried
// until the destination is full.
type dripReader struct {
	n byte
}

func (d *dripReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	d.n++
	p[0] = d.n

	return 1, nil
}

// stallReader is a legal io.Reader that never makes progress: it returns zero
// bytes with a nil error, which io.Reader permits and io.ReadFull would retry
// forever.
type stallReader struct{}

func (stallReader) Read(_ []byte) (int, error) {
	return 0, nil
}

// constReader always succeeds and always yields the same byte. With a value above
// the rejection limit it makes RandString's sampler reject every byte it ever
// sees, which would loop forever without a stalled-pass bound.
type constReader struct {
	v byte
}

func (c constReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = c.v
	}

	return len(p), nil
}

// countingReader records how many times it is called, so that a test can assert
// the reader is not touched at all.
type countingReader struct {
	calls atomic.Int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	c.calls.Add(1)

	for i := range p {
		p[i] = 0x01
	}

	return len(p), nil
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

	// Short reads are retried, not truncated. This reader delivers one byte and then
	// reports EOF, so the buffer can never be filled and the end-of-input surfaces
	// as io.ErrUnexpectedEOF rather than as a short slice of randomness.
	rs := New(&shortReader{})

	b, err = rs.RandomBytes(4)

	require.Error(t, err)
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	require.Nil(t, b)
}

// TestRandomBytes_ShortReadsAreRetried is the other half of the short-read
// contract: a reader that keeps returning one byte at a time with a nil error is
// retried until the destination is full, and must succeed.
func TestRandomBytes_ShortReadsAreRetried(t *testing.T) {
	t.Parallel()

	r := New(&dripReader{})

	b, err := r.RandomBytes(8)

	require.NoError(t, err)
	require.Equal(t, []byte{1, 2, 3, 4, 5, 6, 7, 8}, b)
}

// TestReaderNoProgress covers the readers that would otherwise spin forever.
func TestReaderNoProgress(t *testing.T) {
	t.Parallel()

	t.Run("RandomBytes with a reader that never advances", func(t *testing.T) {
		t.Parallel()

		b, err := New(&stallReader{}).RandomBytes(8)

		require.ErrorIs(t, err, ErrReaderNoProgress)
		require.Nil(t, b)
	})

	t.Run("RandString with a reader that never advances", func(t *testing.T) {
		t.Parallel()

		s, err := New(&stallReader{}).RandString(8)

		require.ErrorIs(t, err, ErrReaderNoProgress)
		require.Empty(t, s)
	})

	t.Run("RandString with a reader whose bytes are always rejected", func(t *testing.T) {
		t.Parallel()

		// The default map has 89 entries, so limit is 178 and every 0xFF byte is
		// rejected. The reader succeeds on every call, so only the stalled-pass
		// bound stops this from looping forever.
		s, err := New(constReader{v: 0xFF}).RandString(8)

		require.ErrorIs(t, err, ErrReaderNoProgress)
		require.Empty(t, s)
	})

	t.Run("the non-failing helpers fall back instead of hanging", func(t *testing.T) {
		t.Parallel()

		r := New(&stallReader{})

		require.NotPanics(t, func() {
			_ = r.RandUint32()
			_ = r.RandUint64()

			u := r.UUIDv7()

			require.Equal(t, byte(0x70), u[6]&0xF0)
			require.Equal(t, byte(0x80), u[8]&0xC0)
		})
	})
}

// TestRandString_InvalidCharMap covers the guard that keeps a degenerate character
// map from dividing by zero or rejecting every byte forever. New and
// WithByteToCharMap both normalize the map, so this is reachable only through the
// zero value, which is documented as unusable.
func TestRandString_InvalidCharMap(t *testing.T) {
	t.Parallel()

	var r Rnd

	s, err := r.RandString(8)

	require.ErrorIs(t, err, ErrInvalidCharMap)
	require.Empty(t, s)
}

// TestZeroLength asserts that a zero length is an empty result rather than an
// error, and that it never touches the reader.
func TestZeroLength(t *testing.T) {
	t.Parallel()

	cr := &countingReader{}
	r := New(cr)

	b, err := r.RandomBytes(0)

	require.NoError(t, err)
	require.Empty(t, b)

	s, err := r.RandString(0)

	require.NoError(t, err)
	require.Empty(t, s)

	require.Zero(t, cr.calls.Load(), "a zero length must not read any entropy")
}

// TestUsesConfiguredReader asserts that the helpers actually draw from the reader
// they were given. Without this, RandUint32/RandUint64 could ignore the reader and
// return math/rand/v2 values unconditionally and every other test would pass.
func TestUsesConfiguredReader(t *testing.T) {
	t.Parallel()

	entropy := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	t.Run("RandomBytes", func(t *testing.T) {
		t.Parallel()

		b, err := New(bytes.NewReader(entropy)).RandomBytes(8)

		require.NoError(t, err)
		require.Equal(t, entropy, b)
	})

	t.Run("RandUint32", func(t *testing.T) {
		t.Parallel()

		got := New(bytes.NewReader(entropy)).RandUint32()

		require.Equal(t, binary.LittleEndian.Uint32(entropy[:4]), got)
	})

	t.Run("RandUint64", func(t *testing.T) {
		t.Parallel()

		got := New(bytes.NewReader(entropy)).RandUint64()

		require.Equal(t, binary.LittleEndian.Uint64(entropy), got)
	})
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

// TestRandStringNoModuloBias is the test that actually detects modulo bias.
//
// TestRandStringUniformSelection above cannot: with a sequential 0..255 reader the
// rejected bytes are exactly limit..255, whose residues mod cmlen are 0,1,2,...,
// and the refill bytes that replace them are 0,1,2,... with the same residues. A
// biased implementation therefore produces byte-identical output, by construction.
//
// So drive it from the real reader instead and check the distribution. A 129-entry
// map is the worst case: limit is 129, so 127 of the 256 byte values are rejected.
// Dropping the rejection would give residues 0..126 exactly twice the weight of
// residues 127 and 128 — a 2x bias that a chi-square test sees immediately.
func TestRandStringNoModuloBias(t *testing.T) {
	t.Parallel()

	const (
		mapLen  = 129
		perChar = 3000
		total   = mapLen * perChar

		// chi-square with 128 degrees of freedom has mean 128 and standard deviation
		// 16, so a uniform sampler stays far below this; the 2x bias above lands
		// around 1500.
		threshold = 300.0
	)

	cm := make([]byte, mapLen)
	for i := range cm {
		cm[i] = byte(i)
	}

	r := New(nil, WithByteToCharMap(cm))

	s, err := r.RandString(total)

	require.NoError(t, err)
	require.Len(t, s, total)

	var counts [mapLen]int

	for i := range total {
		c := s[i]

		require.Less(t, int(c), mapLen, "output byte outside the character map")

		counts[c]++
	}

	expected := float64(perChar)
	chi2 := 0.0

	for _, got := range counts {
		d := float64(got) - expected
		chi2 += d * d / expected
	}

	require.False(t, math.IsNaN(chi2))
	require.Less(t, chi2, threshold,
		"character distribution is not uniform (chi-square %.1f over %d categories): modulo bias", chi2, mapLen)
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
