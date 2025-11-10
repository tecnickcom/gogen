package logutil

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_DefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	require.NotNil(t, cfg)
	require.Equal(t, cfg.Out, os.Stderr)
	require.Equal(t, FormatJSON, cfg.Format)
	require.Equal(t, LevelInfo, cfg.Level)
	require.Empty(t, cfg.CommonAttr)
	require.Nil(t, cfg.LevelHookFn)
}
