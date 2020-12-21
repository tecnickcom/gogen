// +build unit

package logging

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_defaultConfig(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	require.NotNil(t, cfg)
	require.NotEqual(t, 0, cfg.format)
	require.NotEqual(t, 0, cfg.level)
	require.NotEmpty(t, cfg.outputPaths)
	require.NotEmpty(t, cfg.errorOutputPaths)
}
