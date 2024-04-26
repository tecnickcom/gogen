package passwordhash

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithKeyLen(t *testing.T) {
	t.Parallel()

	c := defaultParams()

	var v uint32 = 13

	WithKeyLen(v)(c)
	require.Equal(t, v, c.KeyLen)
}

func TestWithSaltLen(t *testing.T) {
	t.Parallel()

	c := defaultParams()

	var v uint32 = 13

	WithSaltLen(v)(c)
	require.Equal(t, v, c.SaltLen)
}

func TestWithTime(t *testing.T) {
	t.Parallel()

	c := defaultParams()

	var v uint32 = 13

	WithTime(v)(c)
	require.Equal(t, v, c.Time)
}

func TestWithMemory(t *testing.T) {
	t.Parallel()

	c := defaultParams()

	var v uint32 = 13

	WithMemory(v)(c)
	require.Equal(t, v, c.Memory)
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

func TestWithMaxPasswordLength(t *testing.T) {
	t.Parallel()

	c := defaultParams()

	var v uint32 = 13

	WithMaxPasswordLength(v)(c)
	require.Equal(t, v, c.maxPLen)
}
