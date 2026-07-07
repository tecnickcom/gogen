package mysqllock

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWithKeepAliveErrorHandler(t *testing.T) {
	t.Parallel()

	called := false
	locker := New(nil, WithKeepAliveErrorHandler(func(error) { called = true }))

	require.NotNil(t, locker.keepAliveErrHandler)

	locker.keepAliveErrHandler(errors.New("boom"))
	require.True(t, called)
}

func TestOptions(t *testing.T) {
	t.Parallel()

	// Defaults when no options are given.
	def := New(nil)
	require.Equal(t, defaultKeepAliveInterval, def.keepAliveInterval)
	require.Equal(t, defaultKeepAlivePingTimeout, def.keepAlivePingTimeout)
	require.Equal(t, defaultReleaseTimeout, def.releaseTimeout)

	// Positive values are applied.
	set := New(nil,
		WithKeepAliveInterval(5*time.Second),
		WithKeepAlivePingTimeout(2*time.Second),
		WithReleaseTimeout(3*time.Second),
	)
	require.Equal(t, 5*time.Second, set.keepAliveInterval)
	require.Equal(t, 2*time.Second, set.keepAlivePingTimeout)
	require.Equal(t, 3*time.Second, set.releaseTimeout)

	// Non-positive values are ignored and the defaults are kept.
	ignored := New(nil,
		WithKeepAliveInterval(0),
		WithKeepAlivePingTimeout(-time.Second),
		WithReleaseTimeout(-time.Second),
	)
	require.Equal(t, defaultKeepAliveInterval, ignored.keepAliveInterval)
	require.Equal(t, defaultKeepAlivePingTimeout, ignored.keepAlivePingTimeout)
	require.Equal(t, defaultReleaseTimeout, ignored.releaseTimeout)
}
