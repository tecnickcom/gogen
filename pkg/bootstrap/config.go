package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/tecnickcom/gogen/pkg/logsrv"
	"github.com/tecnickcom/gogen/pkg/logutil"
	"github.com/tecnickcom/gogen/pkg/metrics"
)

// Exported sentinel errors returned by Bootstrap and its configuration validation
// so callers can match them with errors.Is.
var (
	// ErrNilBindFunc is returned when Bootstrap is called with a nil BindFunc.
	ErrNilBindFunc = errors.New("bindFn is required")

	// ErrNilContext is returned when the application context is nil.
	ErrNilContext = errors.New("context is required")

	// ErrNilCreateLoggerFunc is returned when the logger factory is nil.
	ErrNilCreateLoggerFunc = errors.New("createLoggerFunc is required")

	// ErrNilLogConfig is returned when WithLogConfig was used with a nil logutil.Config.
	ErrNilLogConfig = errors.New("logConfig is required when using WithLogConfig")

	// ErrNilCreateMetricsClientFunc is returned when the metrics client factory is nil.
	ErrNilCreateMetricsClientFunc = errors.New("createMetricsClientFunc is required")

	// ErrInvalidShutdownTimeout is returned when the shutdown timeout is not positive.
	ErrInvalidShutdownTimeout = errors.New("invalid shutdownTimeout")

	// ErrNilShutdownWaitGroup is returned when the shutdown WaitGroup is nil.
	ErrNilShutdownWaitGroup = errors.New("shutdownWaitGroup is required")

	// ErrNilShutdownSignalChan is returned when the shutdown signal channel is nil.
	ErrNilShutdownSignalChan = errors.New("shutdownSignalChan is required")

	// ErrShutdownTimeout is wrapped into the error returned by Bootstrap when the
	// registered dependants do not finish within the configured shutdown timeout.
	ErrShutdownTimeout = errors.New("graceful shutdown timed out")
)

// CreateLoggerFunc constructs the root logger used by Bootstrap.
type CreateLoggerFunc func() *slog.Logger

// CreateMetricsClientFunc constructs the metrics backend used by Bootstrap.
type CreateMetricsClientFunc func() (metrics.Client, error)

// BindFunc wires application components using the prepared context, logger, and metrics client.
type BindFunc func(context.Context, *slog.Logger, metrics.Client) error

// config stores runtime settings used by Bootstrap.
type config struct {
	// context is the application context.
	context context.Context //nolint:containedctx

	// logConfig stores the logger configuration
	logConfig *logutil.Config

	// logConfigSet records that WithLogConfig was used, so validate can reject
	// a nil logConfig before logger creation dereferences it.
	logConfigSet bool

	// createLoggerFunc is the function used to create a new logger.
	createLoggerFunc CreateLoggerFunc

	// createMetricsClientFunc  is the function used to create a new metrics client.
	createMetricsClientFunc CreateMetricsClientFunc

	// shutdownTimeout is the maximum duration to wait for shutdown.
	shutdownTimeout time.Duration

	// shutdownWaitGroup is used to wait for all goroutines to finish during shutdown.
	shutdownWaitGroup *sync.WaitGroup

	// shutdownSignalChan is used to signal the shutdown event.
	shutdownSignalChan chan struct{}
}

// defaultConfig returns the baseline bootstrap configuration.
//
// It provides production-safe defaults so callers can start with zero options
// and override only the pieces they need.
func defaultConfig() *config {
	return &config{
		context:                 context.Background(),
		logConfig:               nil,
		createLoggerFunc:        defaultCreateLogger,
		createMetricsClientFunc: defaultCreateMetricsClientFunc,
		shutdownTimeout:         30 * time.Second,
		shutdownWaitGroup:       &sync.WaitGroup{},
		shutdownSignalChan:      make(chan struct{}),
	}
}

// defaultCreateLogger returns the process default slog logger.
//
// This keeps bootstrap usable without mandatory logger setup.
func defaultCreateLogger() *slog.Logger {
	return slog.Default()
}

// defaultCreateMetricsClientFunc returns the default no-op metrics client.
//
// It allows bootstrap to run even when no external metrics backend is
// configured, while preserving the same metrics.Client integration points.
func defaultCreateMetricsClientFunc() (metrics.Client, error) {
	return &metrics.Default{}, nil
}

// newLogger builds a slog logger from c.logConfig.
//
// It is selected automatically when WithLogConfig is used.
func (c *config) newLogger() *slog.Logger {
	return logsrv.NewLogger(c.logConfig)
}

// logConfigWithMetricHook returns a copy of c.logConfig whose HookFn first
// increments a per-level metric counter on m and then chains any caller-installed
// hook, giving instant observability into log rates by level.
//
// It works on a shallow copy so the caller-owned logutil.Config is never mutated.
// When no log config was supplied (WithLogConfig unused) it returns nil unchanged;
// in that case the configured logger factory does not consult it.
func (c *config) logConfigWithMetricHook(m metrics.Client) *logutil.Config {
	if c.logConfig == nil {
		return nil
	}

	logCfg := *c.logConfig
	callerHookFn := logCfg.HookFn

	logCfg.HookFn = func(level logutil.LogLevel, message string) {
		m.IncLogLevelCounter(logutil.LevelName(level))

		if callerHookFn != nil {
			callerHookFn(level, message)
		}
	}

	return &logCfg
}

// validate checks that required configuration fields are usable.
//
// It fails fast before service startup so invalid lifecycle dependencies are
// caught early rather than during shutdown-critical paths.
func (c *config) validate() error {
	if c.context == nil {
		return ErrNilContext
	}

	if c.createLoggerFunc == nil {
		return ErrNilCreateLoggerFunc
	}

	if c.logConfigSet && c.logConfig == nil {
		return ErrNilLogConfig
	}

	if c.createMetricsClientFunc == nil {
		return ErrNilCreateMetricsClientFunc
	}

	if c.shutdownTimeout <= 0 {
		return ErrInvalidShutdownTimeout
	}

	if c.shutdownWaitGroup == nil {
		return ErrNilShutdownWaitGroup
	}

	if c.shutdownSignalChan == nil {
		return ErrNilShutdownSignalChan
	}

	return nil
}
