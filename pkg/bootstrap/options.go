package bootstrap

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/tecnickcom/gogen/pkg/logutil"
)

// Option is a type alias for a function that configures the application logger.
type Option func(*config)

// WithContext overrides the application context (useful for testing).
func WithContext(ctx context.Context) Option {
	return func(cfg *config) {
		cfg.context = ctx
	}
}

// WithLogConfig sets the log configuration options.
// This should be used in alternative to WithLogger and WithCreateLoggerFunc.
func WithLogConfig(c *logutil.Config) Option {
	return func(cfg *config) {
		cfg.logConfig = c
		cfg.createLoggerFunc = cfg.newLogger
	}
}

// WithLogger overrides the default application logger.
// This should be used in alternative to WithLogConfig and WithCreateLoggerFunc.
func WithLogger(l *slog.Logger) Option {
	return func(cfg *config) {
		cfg.createLoggerFunc = func() *slog.Logger {
			return l
		}
	}
}

// WithCreateLoggerFunc overrides the root logger creation function.
// This should be used in alternative to WithLogConfig and WithLogger.
func WithCreateLoggerFunc(fn CreateLoggerFunc) Option {
	return func(cfg *config) {
		cfg.createLoggerFunc = fn
	}
}

// WithCreateMetricsClientFunc overrides the default metrics client register.
func WithCreateMetricsClientFunc(fn CreateMetricsClientFunc) Option {
	return func(cfg *config) {
		cfg.createMetricsClientFunc = fn
	}
}

// WithShutdownTimeout sets the shutdown timeout.
// This is the time to wait on exit for a graceful shutdown.
func WithShutdownTimeout(timeout time.Duration) Option {
	return func(cfg *config) {
		cfg.shutdownTimeout = timeout
	}
}

// WithShutdownWaitGroup sets the shared waiting group to communicate externally when the server is shutdown.
// On shutdown Bootstrap will wait for the wg to be zero or the shutdownTimeout to be reached before returning.
// Dependants (e.g. HTTP servers) should increment this group when they start, and reset it when their shutdown process is completed.
func WithShutdownWaitGroup(wg *sync.WaitGroup) Option {
	return func(cfg *config) {
		cfg.shutdownWaitGroup = wg
	}
}

// WithShutdownSignalChan sets the shared channel uset to signal a shutdown.
// The channel is set when the shutdown process starts.
// Dependants (e.g. HTTP servers) should read this channel and start their shutdown process.
func WithShutdownSignalChan(ch chan struct{}) Option {
	return func(cfg *config) {
		cfg.shutdownSignalChan = ch
	}
}
