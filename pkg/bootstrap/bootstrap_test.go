package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/gogen/pkg/logutil"
	"github.com/tecnickcom/gogen/pkg/metrics"
	"github.com/tecnickcom/gogen/pkg/metrics/prometheus"
)

//nolint:gocognit,paralleltest
func TestBootstrap(t *testing.T) {
	shutdownWG := &sync.WaitGroup{}
	shutdownSG := make(chan struct{})

	tests := []struct {
		opts                    []Option
		name                    string
		bindFunc                BindFunc
		createLoggerFunc        CreateLoggerFunc
		createMetricsClientFunc CreateMetricsClientFunc
		stopAfter               time.Duration
		sigterm                 bool
		wantErr                 bool
	}{
		{
			name: "fail with invalid config",
			opts: []Option{
				WithShutdownTimeout(0),
			},
			wantErr: true,
		},
		{
			name: "fail with nil log config",
			opts: []Option{
				WithLogConfig(nil),
			},
			wantErr: true,
		},
		{
			name: "should fail due to create metrics function",
			opts: []Option{
				WithShutdownTimeout(1 * time.Millisecond),
			},
			createMetricsClientFunc: func() (metrics.Client, error) {
				return nil, errors.New("metrics error")
			},
			wantErr: true,
		},
		{
			name: "should fail due to bind function",
			opts: []Option{
				WithShutdownTimeout(1 * time.Millisecond),
			},
			bindFunc: func(context.Context, *slog.Logger, metrics.Client) error {
				return errors.New("bind error")
			},
			wantErr: true,
		},
		{
			name: "should succeed and exit with context cancel",
			opts: []Option{
				WithShutdownTimeout(100 * time.Millisecond),
			},
			bindFunc: func(context.Context, *slog.Logger, metrics.Client) error {
				return nil
			},
			stopAfter: 500 * time.Millisecond,
			wantErr:   false,
		},
		{
			name: "should succeed and exit with SIGTERM",
			opts: []Option{
				WithLogConfig(logutil.DefaultConfig()),
				WithShutdownTimeout(1 * time.Millisecond),
				WithShutdownWaitGroup(shutdownWG),
				WithShutdownSignalChan(shutdownSG),
			},
			bindFunc: func(context.Context, *slog.Logger, metrics.Client) error {
				return nil
			},
			stopAfter: 500 * time.Millisecond,
			sigterm:   true,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// cannot run in parallel because signals are received by all parallel tests
			ctx := t.Context()

			if tt.stopAfter != 0 {
				if tt.sigterm {
					f := func() {
						_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
					}
					time.AfterFunc(tt.stopAfter, f)
				} else {
					var stop context.CancelFunc

					ctx, stop = context.WithTimeout(ctx, tt.stopAfter)
					defer stop()
				}
			}

			opts := []Option{
				WithContext(ctx),
			}
			opts = append(opts, tt.opts...)

			if tt.createMetricsClientFunc != nil {
				opts = append(opts, WithCreateMetricsClientFunc(tt.createMetricsClientFunc))
			} else {
				fn := func() (metrics.Client, error) {
					return prometheus.New()
				}
				opts = append(opts, WithCreateMetricsClientFunc(fn))
			}

			err := Bootstrap(tt.bindFunc, opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Bootstrap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

//nolint:paralleltest // cannot run in parallel because signals are received by all parallel tests
func TestBootstrap_cancelsContextBeforeShutdownWait(t *testing.T) {
	wg := &sync.WaitGroup{}
	sigc := make(chan struct{})

	ctxCanceled := make(chan struct{})

	bindFn := func(ctx context.Context, _ *slog.Logger, _ metrics.Client) error {
		wg.Go(func() {
			// This worker tears down only on ctx.Done(): Bootstrap must cancel
			// the application context before waiting on the shutdown
			// WaitGroup, otherwise this goroutine would stall the whole
			// shutdown timeout.
			<-ctx.Done()
			close(ctxCanceled)
		})

		return nil
	}

	time.AfterFunc(200*time.Millisecond, func() {
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	})

	start := time.Now()

	err := Bootstrap(
		bindFn,
		WithContext(t.Context()),
		WithShutdownTimeout(5*time.Second),
		WithShutdownWaitGroup(wg),
		WithShutdownSignalChan(sigc),
	)
	require.NoError(t, err)

	select {
	case <-ctxCanceled:
	default:
		t.Fatal("the application context was not canceled during shutdown")
	}

	require.Less(t, time.Since(start), 3*time.Second,
		"shutdown must complete before the timeout because ctx is canceled before waiting on the WaitGroup")
}

//nolint:paralleltest // cannot run in parallel because signals are received by all parallel tests
func TestBootstrap_wrapsCallerLogHook(t *testing.T) {
	var callerHookFired atomic.Bool

	callerHook := func(_ logutil.LogLevel, _ string) {
		callerHookFired.Store(true)
	}

	logCfg := logutil.DefaultConfig()
	logCfg.Format = logutil.FormatNone
	logCfg.HookFn = callerHook

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	time.AfterFunc(100*time.Millisecond, cancel)

	bindFn := func(context.Context, *slog.Logger, metrics.Client) error {
		return nil
	}

	err := Bootstrap(
		bindFn,
		WithContext(ctx),
		WithLogConfig(logCfg),
		WithShutdownTimeout(1*time.Second),
	)
	require.NoError(t, err)

	require.True(t, callerHookFired.Load(), "the caller-installed HookFn must keep firing alongside the metric hook")
	require.Equal(t, reflect.ValueOf(callerHook).Pointer(), reflect.ValueOf(logCfg.HookFn).Pointer(),
		"Bootstrap must not mutate the caller-owned logutil.Config")
}

// closeTrackingMetrics is a metrics.Client stub recording Close invocations.
type closeTrackingMetrics struct {
	metrics.Default

	closed   atomic.Bool
	closeErr error
}

func (c *closeTrackingMetrics) Close() error {
	c.closed.Store(true)

	return c.closeErr
}

//nolint:paralleltest // cannot run in parallel because signals are received by all parallel tests
func TestBootstrap_closesMetricsClientOnBindError(t *testing.T) {
	m := &closeTrackingMetrics{
		closeErr: errors.New("close error"), // also exercises the close-error logging path
	}

	bindFn := func(context.Context, *slog.Logger, metrics.Client) error {
		return errors.New("bind error")
	}

	err := Bootstrap(
		bindFn,
		WithContext(t.Context()),
		WithShutdownTimeout(1*time.Second),
		WithCreateMetricsClientFunc(func() (metrics.Client, error) { return m, nil }),
	)
	require.Error(t, err)
	require.True(t, m.closed.Load(), "the metrics client must be closed on the bindFn error path")
}

//nolint:paralleltest // cannot run in parallel because signals are received by all parallel tests
func TestBootstrap_closesMetricsClientOnShutdown(t *testing.T) {
	m := &closeTrackingMetrics{}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	time.AfterFunc(100*time.Millisecond, cancel)

	bindFn := func(context.Context, *slog.Logger, metrics.Client) error {
		return nil
	}

	err := Bootstrap(
		bindFn,
		WithContext(ctx),
		WithShutdownTimeout(1*time.Second),
		WithCreateMetricsClientFunc(func() (metrics.Client, error) { return m, nil }),
	)
	require.NoError(t, err)
	require.True(t, m.closed.Load(), "the metrics client must be closed on shutdown")
}

func Test_syncWaitGroupTimeout(t *testing.T) {
	t.Parallel()

	wg := &sync.WaitGroup{}

	wg.Add(1)

	// timeout
	syncWaitGroupTimeout(wg, 1*time.Millisecond, slog.Default())

	wg.Add(-1)

	// wait complete
	syncWaitGroupTimeout(wg, 1*time.Second, slog.Default())
}
