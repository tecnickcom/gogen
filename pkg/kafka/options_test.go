package kafka

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_WithSessionTimeout(t *testing.T) {
	t.Parallel()

	v := time.Second * 17

	cfg := &config{}
	WithSessionTimeout(v)(cfg)
	require.Equal(t, v, cfg.sessionTimeout)
}
