package sfcache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWithTTLFunc(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, 1, 1*time.Minute)
	require.Nil(t, c.ttlFn)

	ttlFn := func(_ string, _ any) time.Duration {
		return 1 * time.Second
	}

	WithTTLFunc(ttlFn)(c)
	require.NotNil(t, c.ttlFn)
	require.Equal(t, 1*time.Second, c.ttlFn("example.com", nil))

	WithTTLFunc[string, any](nil)(c)
	require.Nil(t, c.ttlFn)
}

func TestWithStaleIfError(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, 1, 1*time.Minute)
	require.Equal(t, time.Duration(0), c.maxStale)

	WithStaleIfError[string, any](30 * time.Second)(c)
	require.Equal(t, 30*time.Second, c.maxStale)

	WithStaleIfError[string, any](0)(c)
	require.Equal(t, time.Duration(0), c.maxStale)
}
