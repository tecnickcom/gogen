package bootstrap

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/Vonage/gosrvlib/pkg/metrics"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWithContext(t *testing.T) {
	t.Parallel()

	v := context.WithValue(context.Background(), struct{}{}, "")
	cfg := &config{}
	WithContext(v)(cfg)
	require.Equal(t, v, cfg.context)
}

func TestWithLogger(t *testing.T) {
	t.Parallel()

	l := zap.NewNop()
	cfg := &config{}
	WithLogger(l)(cfg)
	require.NotNil(t, cfg.createLoggerFunc)

	ll, err := cfg.createLoggerFunc()
	require.NoError(t, err)
	require.Equal(t, l, ll)
}

func TestWithCreateLoggerFunc(t *testing.T) {
	t.Parallel()

	v := func() (*zap.Logger, error) {
		return nil, nil
	}
	cfg := &config{}
	WithCreateLoggerFunc(v)(cfg)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(cfg.createLoggerFunc).Pointer())
}

func TestWithCreateMetricsClientFunc(t *testing.T) {
	t.Parallel()

	v := func() (metrics.Client, error) {
		return nil, nil
	}
	cfg := &config{}
	WithCreateMetricsClientFunc(v)(cfg)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(cfg.createMetricsClientFunc).Pointer())
}

func TestWithShutdownTimeout(t *testing.T) {
	t.Parallel()

	v := 17 * time.Second
	cfg := defaultConfig()
	WithShutdownTimeout(v)(cfg)
	require.Equal(t, v, cfg.shutdownTimeout)
}

func TestWithShutdownWaitGroup(t *testing.T) {
	t.Parallel()

	v := &sync.WaitGroup{}
	cfg := defaultConfig()
	WithShutdownWaitGroup(v)(cfg)
	require.Equal(t, v, cfg.shutdownWaitGroup)
}
