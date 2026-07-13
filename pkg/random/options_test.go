package random

import (
	"errors"
	"sync/atomic"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/require"
)

func TestWithByteToCharMap(t *testing.T) {
	t.Parallel()

	want := []byte(chrMapDefault)
	c := &Rnd{}

	WithByteToCharMap(want)(c)
	require.Equal(t, want, c.chrMap)

	WithByteToCharMap(nil)(c)
	require.Equal(t, want, c.chrMap)

	WithByteToCharMap([]byte{})(c)
	require.Equal(t, want, c.chrMap)

	WithByteToCharMap([]byte("0123456789abcdefx"))(c)
	require.Len(t, c.chrMap, 17)

	// The literal 256 is deliberate: asserting against chrMapMaxLen would re-use the
	// constant under test, so raising it could never fail this test — and a map
	// longer than 256 bytes cannot be addressed by a single byte.
	longMap := make([]byte, chrMapMaxLen+1)

	WithByteToCharMap(longMap)(c)
	require.Len(t, c.chrMap, 256)
}

// TestWithByteToCharMap_CopiesInput asserts the generator does not alias the
// caller's slice. Retaining it would let a later mutation silently reconfigure the
// alphabet — clearing the buffer, the usual "wipe my secret" idiom, would turn
// RandString into a zero-entropy NUL generator — and a concurrent mutation would
// be a data race.
func TestWithByteToCharMap_CopiesInput(t *testing.T) {
	t.Parallel()

	cm := []byte("ABCD")

	r := New(nil, WithByteToCharMap(cm))

	before, err := r.RandString(64)
	require.NoError(t, err)

	// The caller mutates and then clears the slice it handed over.
	for i := range cm {
		cm[i] = '!'
	}

	clear(cm)

	after, err := r.RandString(64)
	require.NoError(t, err)

	require.NotContains(t, after, "!", "generator must not see the caller's mutation")
	require.NotContains(t, after, "\x00", "clearing the caller's slice must not zero the alphabet")

	for i := range after {
		require.Contains(t, "ABCD", string(after[i]), "alphabet must still be the original one")
	}

	require.NotEqual(t, before, after, "output must still be random")

	// Truncation must copy too: it re-slices the caller's backing array.
	long := make([]byte, chrMapMaxLen+1)
	for i := range long {
		long[i] = 'Z'
	}

	rl := New(nil, WithByteToCharMap(long))

	clear(long)

	s, err := rl.RandString(16)
	require.NoError(t, err)
	require.Equal(t, "ZZZZZZZZZZZZZZZZ", s, "truncated map must be a copy as well")
}

// TestWithFallbackHook asserts the hook fires whenever a non-failing helper
// silently substitutes math/rand/v2 for a failing reader.
func TestWithFallbackHook(t *testing.T) {
	t.Parallel()

	t.Run("fires for every non-failing helper", func(t *testing.T) {
		t.Parallel()

		var calls atomic.Int64

		r := New(
			iotest.ErrReader(errors.New("test-fallback-hook")),
			WithFallbackHook(func() { calls.Add(1) }),
		)

		_ = r.RandUint32()

		require.Equal(t, int64(1), calls.Load(), "RandUint32")

		_ = r.RandUint64()

		require.Equal(t, int64(2), calls.Load(), "RandUint64")

		_ = r.UUIDv7()

		require.Equal(t, int64(3), calls.Load(), "UUIDv7")
	})

	t.Run("does not fire when the reader works", func(t *testing.T) {
		t.Parallel()

		var calls atomic.Int64

		r := New(nil, WithFallbackHook(func() { calls.Add(1) }))

		_ = r.RandUint64()
		_ = r.UUIDv7()

		require.Zero(t, calls.Load(), "the default reader cannot fail, so the hook must never fire")
	})

	t.Run("the fallback stays silent without a hook", func(t *testing.T) {
		t.Parallel()

		r := New(iotest.ErrReader(errors.New("test-no-hook")))

		require.NotPanics(t, func() {
			_ = r.RandUint64()
			_ = r.UUIDv7()
		})
	})
}
