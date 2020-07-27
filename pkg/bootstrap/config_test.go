// +build unit

package bootstrap

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_defaultConfig(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.context)
	require.NotNil(t, cfg.createLoggerFunc)
}

func Test_defaultCreateLogger(t *testing.T) {
	t.Parallel()

	l, err := defaultCreateLogger()
	require.NotNil(t, l)
	require.NoError(t, err)
}
