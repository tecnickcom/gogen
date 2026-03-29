package bootstrap

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/tecnickcom/gogen/pkg/logutil"
)

// Option configures Bootstrap runtime behavior.
type Option func(*config)

// WithContext overrides the root application context used by Bootstrap.
//
// This is especially useful in tests and controlled runtime environments where
// cancellation and deadlines are managed externally.
func WithContext(ctx context.Context) Option {
	return func(cfg *config) {
		cfg.context = ctx
	}
}

// WithLogConfig configures logger creation from a logutil.Config.
//
// It should be used as an alternative to WithLogger and
// WithCreateLoggerFunc. The benefit is centralized logger policy with optional
// hooks (including log-level metrics wiring performed by Bootstrap).
func WithLogConfig(c *logutil.Config) Option {
	return func(cfg *config) {
		cfg.logConfig = c
		cfg.createLoggerFunc = cfg.newLogger
	}
}

// WithLogger injects a prebuilt root logger instance.
//
// It should be used as an alternative to WithLogConfig and
// WithCreateLoggerFunc when logger construction is owned by the caller.
func WithLogger(l *slog.Logger) Option {
	return func(cfg *config) {
		cfg.createLoggerFunc = func() *slog.Logger {
			return l
		}
	}
}

// WithCreateLoggerFunc injects a custom logger factory.
//
// It should be used as an alternative to WithLogConfig and WithLogger when a
// fresh logger instance must be created lazily at bootstrap time.
func WithCreateLoggerFunc(fn CreateLoggerFunc) Option {
	return func(cfg *config) {
		cfg.createLoggerFunc = fn
	}
}

// WithCreateMetricsClientFunc overrides metrics client creation.
//
// Use it to plug in a custom metrics backend while keeping Bootstrap lifecycle
// behavior unchanged.
func WithCreateMetricsClientFunc(fn CreateMetricsClientFunc) Option {
	return func(cfg *config) {
		cfg.createMetricsClientFunc = fn
	}
}

// WithShutdownTimeout sets the maximum graceful-shutdown wait duration.
//
// Bootstrap returns after this timeout even if dependant goroutines have not
// completed, preventing indefinite process hangs.
func WithShutdownTimeout(timeout time.Duration) Option {
	return func(cfg *config) {
		cfg.shutdownTimeout = timeout
	}
}

// WithShutdownWaitGroup sets the shared WaitGroup used to track dependant shutdown.
//
// On shutdown, Bootstrap waits until wg reaches zero or the configured timeout
// expires. Dependants (for example HTTP servers) should Add before starting
// and Done when teardown is complete.
func WithShutdownWaitGroup(wg *sync.WaitGroup) Option {
	return func(cfg *config) {
		cfg.shutdownWaitGroup = wg
	}
}

// WithShutdownSignalChan sets the shared broadcast channel for shutdown start.
//
// Bootstrap closes this channel when shutdown begins. Dependants can watch this
// channel to trigger their own graceful termination logic.
func WithShutdownSignalChan(ch chan struct{}) Option {
	return func(cfg *config) {
		cfg.shutdownSignalChan = ch
	}
}
