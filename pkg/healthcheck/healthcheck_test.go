package healthcheck

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testHealthChecker struct {
	delay time.Duration
	err   error
}

func (th *testHealthChecker) HealthCheck(_ context.Context) error {
	if th.delay != 0 {
		time.Sleep(th.delay)
	}

	return th.err
}

func TestNew(t *testing.T) {
	t.Parallel()

	hc := &testHealthChecker{}
	h := New("hc-id_1", hc)
	require.NotNil(t, h)
	require.Equal(t, "hc-id_1", h.ID)
	require.Equal(t, h.Checker, hc)
}
