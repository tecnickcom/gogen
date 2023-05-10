package bootstrap

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Vonage/gosrvlib/pkg/logging"
	"github.com/Vonage/gosrvlib/pkg/metrics"
	"go.uber.org/zap"
)

// CreateLoggerFunc creates a new logger.
type CreateLoggerFunc func() (*zap.Logger, error)

// CreateMetricsClientFunc creates a new metrics client.
type CreateMetricsClientFunc func() (metrics.Client, error)

// BindFunc represents the function responsible to wire up all components of the application.
type BindFunc func(context.Context, *zap.Logger, metrics.Client) error

type config struct {
	context                 context.Context
	createLoggerFunc        CreateLoggerFunc
	createMetricsClientFunc CreateMetricsClientFunc
	shutdownTimeout         time.Duration
	shutdownWaitGroup       *sync.WaitGroup
	shutdownSignalChan      chan struct{}
}

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

func defaultCreateLogger() (*zap.Logger, error) {
	return logging.NewLogger() //nolint:wrapcheck
}

func defaultCreateMetricsClientFunc() (metrics.Client, error) {
	return &metrics.Default{}, nil
}

// validate the configuration.
func (c *config) validate() error {
	if c.context == nil {
		return fmt.Errorf("context is required")
	}

	if c.createLoggerFunc == nil {
		return fmt.Errorf("createLoggerFunc is required")
	}

	if c.createMetricsClientFunc == nil {
		return fmt.Errorf("createMetricsClientFunc is required")
	}

	if c.shutdownTimeout <= 0 {
		return fmt.Errorf("invalid shutdownTimeout")
	}

	if c.shutdownWaitGroup == nil {
		return fmt.Errorf("shutdownWaitGroup is required")
	}

	if c.shutdownSignalChan == nil {
		return fmt.Errorf("shutdownSignalChan is required")
	}

	return nil
}
