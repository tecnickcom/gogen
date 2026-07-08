package passwordhash

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithKeyLen(t *testing.T) {
	t.Parallel()

	c := defaultParams()

	var v uint32 = 20

	WithKeyLen(v)(c)
	require.Equal(t, v, c.KeyLen)
}

func TestWithKeyLen_clamp(t *testing.T) {
	t.Parallel()

	c := defaultParams()

	WithKeyLen(4)(c)
	require.Equal(t, uint32(minHashKeyLen), c.KeyLen)
}

func TestWithKeyLen_clampCeiling(t *testing.T) {
	t.Parallel()

	c := defaultParams()

	WithKeyLen(maxVerifyKeyLen + 1)(c)
	require.Equal(t, uint32(maxVerifyKeyLen), c.KeyLen)
}

func TestWithSaltLen(t *testing.T) {
	t.Parallel()

	c := defaultParams()

	var v uint32 = 13

	WithSaltLen(v)(c)
	require.Equal(t, v, c.SaltLen)
}

func TestWithSaltLen_clamp(t *testing.T) {
	t.Parallel()

	c := defaultParams()

	WithSaltLen(1)(c)
	require.Equal(t, uint32(minHashSaltLen), c.SaltLen)
}

func TestWithSaltLen_clampCeiling(t *testing.T) {
	t.Parallel()

	c := defaultParams()

	WithSaltLen(maxVerifySaltLen + 1)(c)
	require.Equal(t, uint32(maxVerifySaltLen), c.SaltLen)
}

func TestWithTime(t *testing.T) {
	t.Parallel()

	c := defaultParams()

	var v uint32 = 13

	WithTime(v)(c)
	require.Equal(t, v, c.Time)
}

func TestWithTime_clampCeiling(t *testing.T) {
	t.Parallel()

	c := defaultParams()

	WithTime(maxVerifyTime + 1)(c)
	require.Equal(t, uint32(maxVerifyTime), c.Time)
}

func TestWithMemory(t *testing.T) {
	t.Parallel()

	c := defaultParams()

	var v uint32 = 13

	WithMemory(v)(c)
	require.Equal(t, v, c.Memory)
}

func TestWithMemory_clampCeiling(t *testing.T) {
	t.Parallel()

	c := defaultParams()

	WithMemory(maxVerifyMemory + 1)(c)
	require.Equal(t, uint32(maxVerifyMemory), c.Memory)
}

func TestWithThreads(t *testing.T) {
	t.Parallel()

	c := defaultParams()

	var v uint8 = 3

	WithThreads(v)(c)
	require.Equal(t, v, c.Threads)
}

func TestWithMinPasswordLength(t *testing.T) {
	t.Parallel()

	c := defaultParams()

	var v uint32 = 13

	WithMinPasswordLength(v)(c)
	require.Equal(t, v, c.minPLen)
}

func TestWithMinPasswordLength_clamp(t *testing.T) {
	t.Parallel()

	c := defaultParams()

	WithMinPasswordLength(0)(c)
	require.Equal(t, uint32(minPasswordLength), c.minPLen)
}

func TestWithMaxPasswordLength(t *testing.T) {
	t.Parallel()

	c := defaultParams()

	var v uint32 = 13

	WithMaxPasswordLength(v)(c)
	require.Equal(t, v, c.maxPLen)
}
