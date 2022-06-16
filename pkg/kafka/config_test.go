package kafka

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_defaultConfig(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	require.NotNil(t, cfg)
	require.NotEmpty(t, cfg.sessionTimeout)
	require.Equal(t, int64(-1), cfg.startOffset)
}
