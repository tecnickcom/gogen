// Package bootstrap provides a simple way to bootstrap an application with a managed
// logging framework, metrics and application context.
package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Vonage/gosrvlib/pkg/logging"
)

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

	m, err := cfg.createMetricsClientFunc()
	if err != nil {
		return fmt.Errorf("error creating application metric: %w", err)
	}

	l, err := cfg.createLoggerFunc()
	if err != nil {
		return fmt.Errorf("error creating application logger: %w", err)
	}

	l = logging.WithLevelFunctionHook(l, m.IncLogLevelCounter)
	ctx = logging.WithLogger(ctx, l)

	defer logging.Sync(l)

	l.Info("binding application components")

	if err := bindFn(ctx, l, m); err != nil {
		return fmt.Errorf("application bootstrap error: %w", err)
	}

	l.Info("application started")

	done := make(chan struct{})

	// handle shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		defer close(done)

		select {
		case <-quit: // quit on user signal
		case <-ctx.Done(): // context canceled
		}

		// cancel the application context
		l.Debug("shutdown signal received")
		cancel()
	}()

	<-done
	l.Info("application stopped")

	return nil
}
