package bootstrap

import (
	"context"
	"errors"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/Vonage/gosrvlib/pkg/logging"
	"github.com/Vonage/gosrvlib/pkg/metrics"
	"github.com/Vonage/gosrvlib/pkg/metrics/prometheus"
	"github.com/Vonage/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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
		checkLogs               bool
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
			name: "should fail due to create logger function",
			opts: []Option{
				WithShutdownTimeout(1 * time.Millisecond),
			},
			createLoggerFunc: func() (*zap.Logger, error) {
				return nil, errors.New("log error")
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
			bindFunc: func(context.Context, *zap.Logger, metrics.Client) error {
				return errors.New("bind error")
			},
			wantErr: true,
		},
		{
			name: "should succeed and exit with context cancel",
			opts: []Option{
				WithShutdownTimeout(100 * time.Millisecond),
			},
			bindFunc: func(context.Context, *zap.Logger, metrics.Client) error {
				return nil
			},
			stopAfter: 500 * time.Millisecond,
			wantErr:   false,
		},
		{
			name: "should succeed and exit with SIGTERM",
			opts: []Option{
				WithShutdownTimeout(1 * time.Millisecond),
				WithShutdownWaitGroup(shutdownWG),
				WithShutdownSignalChan(shutdownSG),
			},
			bindFunc: func(context.Context, *zap.Logger, metrics.Client) error {
				return nil
			},
			stopAfter: 500 * time.Millisecond,
			sigterm:   true,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// cannot run in parallel because signals are received by all parallel tests
			var ctx context.Context
			ctx, logs := testutil.ContextWithLogObserver(zap.DebugLevel)

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

			if tt.createLoggerFunc != nil {
				opts = append(opts, WithCreateLoggerFunc(tt.createLoggerFunc))
			} else {
				fn := func() (*zap.Logger, error) {
					return logging.FromContext(ctx), nil
				}
				opts = append(opts, WithCreateLoggerFunc(fn))
			}

			if tt.createMetricsClientFunc != nil {
				opts = append(opts, WithCreateMetricsClientFunc(tt.createMetricsClientFunc))
			} else {
				fn := func() (metrics.Client, error) {
					return prometheus.New() //nolint:wrapcheck
				}
				opts = append(opts, WithCreateMetricsClientFunc(fn))
			}

			if err := Bootstrap(tt.bindFunc, opts...); (err != nil) != tt.wantErr {
				t.Errorf("Bootstrap() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.checkLogs {
				entries := logs.All()
				require.Equal(t, "application started", entries[0].Message)
				require.Equal(t, "application stopped", entries[1].Message)
			}
		})
	}
}

func Test_syncWaitGroupTimeout(t *testing.T) {
	t.Parallel()

	wg := &sync.WaitGroup{}

	wg.Add(1)

	// timeout
	syncWaitGroupTimeout(wg, 1*time.Millisecond, logging.NopLogger())

	wg.Add(-1)

	// wait complete
	syncWaitGroupTimeout(wg, 1*time.Second, logging.NopLogger())
}
