package bootstrap

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/tecnickcom/gogen/pkg/logging"
	"github.com/tecnickcom/gogen/pkg/metrics"
	"go.uber.org/zap"
)

// CreateLoggerFunc creates a new logger.
type CreateLoggerFunc func() (*zap.Logger, error)

// CreateMetricsClientFunc creates a new metrics client.
type CreateMetricsClientFunc func() (metrics.Client, error)

// BindFunc represents the function responsible to wire up all components of the application.
type BindFunc func(context.Context, *zap.Logger, metrics.Client) error

// config represents the bootstrap configuration.
type config struct {
	// context is the application context.
	context context.Context //nolint:containedctx

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

// defaultConfig returns the default configuration.
func defaultConfig() *config {
	return &config{
		context:                 context.Background(),
		createLoggerFunc:        defaultCreateLogger,
		createMetricsClientFunc: defaultCreateMetricsClientFunc,
		shutdownTimeout:         30 * time.Second,
		shutdownWaitGroup:       &sync.WaitGroup{},
		shutdownSignalChan:      make(chan struct{}),
	}
}

// defaultCreateLogger creates a default logger.
func defaultCreateLogger() (*zap.Logger, error) {
	return logging.NewLogger() //nolint:wrapcheck
}

// defaultCreateMetricsClient creates a default metrics client.
func defaultCreateMetricsClientFunc() (metrics.Client, error) {
	return &metrics.Default{}, nil
}

// validate the configuration.
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
