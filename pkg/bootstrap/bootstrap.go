/*
Package bootstrap provides a simple way to bootstrap an application with managed
configuration, logging, metrics, application context, and shutdown signals.

For an implementation example see in order:
  - examples/service/cmd/main.go
  - examples/service/internal/cli/cli.go
  - examples/service/internal/cli/bind.go
*/
package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Vonage/gosrvlib/pkg/logging"
	"go.uber.org/zap"
)

// Bootstrap is the function in charge of configuring the core components
// of an application and handling the lifecycle of its context.
func Bootstrap(bindFn BindFunc, opts ...Option) error {
	cfg := defaultConfig()

	for _, applyOpt := range opts {
		applyOpt(cfg)
	}

	if err := cfg.validate(); err != nil {
		return err
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

	l.Debug("binding application components")

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
		case <-quit:
			l.Debug("shutdown signal received")
		case <-ctx.Done():
			l.Warn("context canceled")
		}
	}()

	<-done
	l.Info("application stopping")

	// send shutdown signal to all dependants (e.g. HTTP servers)
	close(cfg.shutdownSignalChan)

	// wait for graceful shutdown of dependants
	syncWaitGroupTimeout(cfg.shutdownWaitGroup, cfg.shutdownTimeout, l)

	// cancel application context
	cancel()

	l.Info("application stopped")

	return nil
}

// syncWaitGroupTimeout adds a timeout to the sync.WaitGroup.Wait().
func syncWaitGroupTimeout(wg *sync.WaitGroup, timeout time.Duration, l *zap.Logger) {
	wait := make(chan struct{})

	go func() {
		defer close(wait)
		wg.Wait()
	}()

	select {
	case <-wait:
		l.Debug("dependands shutdown complete")
	case <-time.After(timeout):
		l.Warn("dependands shutdown timeout")
	}
}
