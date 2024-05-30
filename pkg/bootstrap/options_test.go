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

	cfg := &config{}

	v := context.WithValue(context.Background(), struct{}{}, "")
	WithContext(v)(cfg)
	require.Equal(t, v, cfg.context)
}

func TestWithLogger(t *testing.T) {
	t.Parallel()

	cfg := &config{}

	l := zap.NewNop()
	WithLogger(l)(cfg)
	require.NotNil(t, cfg.createLoggerFunc)

	ll, err := cfg.createLoggerFunc()
	require.NoError(t, err)
	require.Equal(t, l, ll)
}

func TestWithCreateLoggerFunc(t *testing.T) {
	t.Parallel()

	cfg := &config{}

	v := func() (*zap.Logger, error) {
		return nil, nil //nolint:nilnil
	}
	WithCreateLoggerFunc(v)(cfg)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(cfg.createLoggerFunc).Pointer())
}

func TestWithCreateMetricsClientFunc(t *testing.T) {
	t.Parallel()

	cfg := &config{}

	v := func() (metrics.Client, error) {
		return nil, nil //nolint:nilnil
	}
	WithCreateMetricsClientFunc(v)(cfg)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(cfg.createMetricsClientFunc).Pointer())
}

func TestWithShutdownTimeout(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()

	v := 17 * time.Second
	WithShutdownTimeout(v)(cfg)
	require.Equal(t, v, cfg.shutdownTimeout)
}

func TestWithShutdownWaitGroup(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()

	v := &sync.WaitGroup{}
	WithShutdownWaitGroup(v)(cfg)
	require.Equal(t, v, cfg.shutdownWaitGroup)
}

func TestWithShutdownSignalChan(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()

	v := make(chan struct{})
	WithShutdownSignalChan(v)(cfg)
	require.Equal(t, v, cfg.shutdownSignalChan)
}
