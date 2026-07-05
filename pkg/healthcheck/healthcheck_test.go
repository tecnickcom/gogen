package healthcheck

import (
	"context"
	"errors"
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

type panicHealthChecker struct {
	msg string
}

func (ph *panicHealthChecker) HealthCheck(_ context.Context) error {
	panic(ph.msg)
}

func TestNew(t *testing.T) {
	t.Parallel()

	hc := &testHealthChecker{}
	h := New("hc-id_1", hc)
	require.NotNil(t, h)
	require.Equal(t, "hc-id_1", h.ID)
	require.Equal(t, h.Checker, hc)
}

func TestHealthCheckFunc(t *testing.T) {
	t.Parallel()

	called := false

	var ok HealthChecker = HealthCheckFunc(func(_ context.Context) error {
		called = true

		return nil
	})

	require.NoError(t, ok.HealthCheck(t.Context()))
	require.True(t, called)

	sentinel := errors.New("boom")
	failing := HealthCheckFunc(func(_ context.Context) error { return sentinel })
	require.ErrorIs(t, failing.HealthCheck(t.Context()), sentinel)
}
