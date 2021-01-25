// Package bootstrap provides a simple way to bootstrap an application with a managed
// logging framework and application context
package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"github.com/nexmoinc/gosrvlib/pkg/metrics"
	"go.uber.org/zap"
)

// BindFunc represents the function responsible to wire up all components of the application.
type BindFunc func(context.Context, *zap.Logger, *metrics.Client) error

// CreateLoggerFunc creates a new logger.
type CreateLoggerFunc func() (*zap.Logger, error)

// CreateMetricsClientFunc creates a new metrics client.
type CreateMetricsClientFunc func() (*metrics.Client, error)

// Bootstrap is the function in charge of configuring the core components
// of an application and handling the lifecycle of its context.
func Bootstrap(bindFn BindFunc, opts ...Option) error {
	cfg := defaultConfig()
	for _, applyOpt := range opts {
		applyOpt(cfg)
	}

	// create application context
	ctx, cancel := context.WithCancel(cfg.context)
	defer cancel()

	l, err := cfg.createLoggerFunc()
	if err != nil {
		return fmt.Errorf("error creating application logger: %w", err)
	}

	// attach root logger to application context
	ctx = logging.WithLogger(ctx, l)
	defer logging.Sync(l)

	m, err := cfg.createMetricsClientFunc()
	if err != nil {
		return fmt.Errorf("error creating application metric: %w", err)
	}

	l.Info("binding application components")
	if err := bindFn(ctx, l, m); err != nil {
		return fmt.Errorf("application bootstrap error: %w", err)
	}
	l.Info("application started")

	done := make(chan struct{})

	go func() {
		defer close(done)

		// handle shutdown signals
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

		select {
		case <-ctx.Done():
			// context canceled
		case <-quit:
			// quit on user signal
		}

		// cancel the application context
		l.Debug("shutdown signal received")
		cancel()
	}()

	<-done
	l.Info("application stopped")

	return nil
}
