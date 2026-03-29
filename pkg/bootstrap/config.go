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

// validate checks that required configuration fields are usable.
//
// It fails fast before service startup so invalid lifecycle dependencies are
// caught early rather than during shutdown-critical paths.
func (c *config) validate() error {
	if c.context == nil {
		return errors.New("context is required")
	}

	if c.createLoggerFunc == nil {
		return errors.New("createLoggerFunc is required")
	}

	if c.createMetricsClientFunc == nil {
		return errors.New("createMetricsClientFunc is required")
	}

	if c.shutdownTimeout <= 0 {
		return errors.New("invalid shutdownTimeout")
	}

	if c.shutdownWaitGroup == nil {
		return errors.New("shutdownWaitGroup is required")
	}

	if c.shutdownSignalChan == nil {
		return errors.New("shutdownSignalChan is required")
	}

	return nil
}
