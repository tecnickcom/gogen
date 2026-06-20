package kafkacgo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_defaultConfig(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.configMap)
	require.NotNil(t, cfg.messageEncodeFunc)
	require.NotNil(t, cfg.messageDecodeFunc)
	require.Equal(t, defaultFlushTimeoutMs, cfg.flushTimeoutMs)
}

func Test_WithFlushTimeout(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	WithFlushTimeout(7 * time.Second)(cfg)
	require.Equal(t, 7_000, cfg.flushTimeoutMs)
}
