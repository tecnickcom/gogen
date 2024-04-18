package kafkacgo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_defaultConfig(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.configMap)
	require.NotNil(t, cfg.messageEncodeFunc)
	require.NotNil(t, cfg.messageDecodeFunc)
}
