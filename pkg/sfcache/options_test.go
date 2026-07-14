package sfcache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWithTTLFunc(t *testing.T) {
	t.Parallel()

	c := New(nopLookupFn, Config{Size: 1, TTL: 1 * time.Minute})
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

// TestOption_type_inference pins the property Config exists for: every setting
// can be applied without spelling out the cache's type parameters. It is a
// compile-time assertion; the runtime checks are incidental.
func TestOption_type_inference(t *testing.T) {
	t.Parallel()

	ttlFn := func(_ string, _ any) time.Duration {
		return 1 * time.Second
	}

	c := New(
		nopLookupFn,
		Config{
			Size:              2,
			TTL:               1 * time.Minute,
			MaxStale:          30 * time.Second,
			MaxStaleOnFailure: 45 * time.Second,
		},
		WithTTLFunc(ttlFn), // no explicit type arguments anywhere
	)

	require.Equal(t, 2, c.size)
	require.Equal(t, 1*time.Minute, c.ttl)
	require.Equal(t, 30*time.Second, c.maxStale)
	require.Equal(t, 45*time.Second, c.maxStaleOnFailure)
	require.NotNil(t, c.ttlFn)
}
