/*
Package bootstrap solves the repetitive, error-prone problem of wiring together
the core infrastructure of a Go service: context lifecycle, structured logging,
metrics collection, OS signal handling, and graceful shutdown — all with a single
function call.

# Problem

Every long-running Go service needs the same boilerplate: create a context,
configure a logger, initialize metrics, wire up components, listen for SIGTERM,
and coordinate a clean shutdown without dropping in-flight work. Doing this
correctly — with timeouts, WaitGroups, and proper signal handling — is
repetitive and easy to get wrong. This package encapsulates that pattern once,
tested and production-ready.

# How It Works

The entry point is [Bootstrap]. It accepts a [BindFunc] — the caller-supplied
function that wires up all application-specific components — plus a variadic
list of [Option] values that tune the runtime behavior:

 1. A cancellable [context.Context] is created and threaded through the entire
    application via BindFunc.
 2. A [metrics.Client] is created (Prometheus by default) and passed to BindFunc.
 3. A [*slog.Logger] is created and passed to BindFunc.
    If a [logutil.Config] is provided with [WithLogConfig], the logger
    automatically emits a metrics counter for every log line, broken down by
    level — giving instant observability into error rates.
 4. BindFunc is called. This is where the caller registers HTTP servers,
    database connections, background workers, etc.
 5. Bootstrap blocks until it receives os.Interrupt, SIGTERM, or SIGINT, or
    until the context is canceled externally.
 6. A shutdown signal is broadcast on the shared channel
    (see [WithShutdownSignalChan]) so every registered dependent can start its
    own teardown.
 7. Bootstrap waits for all dependants to finish via a [sync.WaitGroup]
    (see [WithShutdownWaitGroup]), bounded by a configurable timeout
    (see [WithShutdownTimeout]) to prevent hanging indefinitely.

# Key Features

  - Single-function API: one call handles the entire lifecycle.
  - Functional options pattern: zero mandatory configuration; override only
    what you need via [WithContext], [WithLogConfig], [WithLogger],
    [WithCreateLoggerFunc], [WithCreateMetricsClientFunc],
    [WithShutdownTimeout], [WithShutdownWaitGroup], and [WithShutdownSignalChan].
  - Automatic log-level metrics: when a [logutil.Config] is supplied, every
    log emission increments a labeled counter, enabling SLO-style alerting on
    error log rates without extra instrumentation.
  - Graceful shutdown with timeout: dependants (HTTP servers, consumers, …)
    communicate completion through a shared [sync.WaitGroup]; Bootstrap honors
    a deadline so a stuck goroutine can never block the process forever.
  - Testable by design: [WithContext] lets tests inject a cancellable context
    to drive the shutdown path without sending real OS signals.

# Usage

Wire your application in a [BindFunc] and pass it to [Bootstrap]:

	func bind(ctx context.Context, l *slog.Logger, m metrics.Client) error {
	    // register HTTP servers, workers, DB connections, etc.
	    return nil
	}

	func main() {
	    shutdownWG := &sync.WaitGroup{}
	    shutdownCh := make(chan struct{})

	    err := bootstrap.Bootstrap(
	        bind,
	        bootstrap.WithLogConfig(logutil.DefaultConfig()),
	        bootstrap.WithShutdownTimeout(30*time.Second),
	        bootstrap.WithShutdownWaitGroup(shutdownWG),
	        bootstrap.WithShutdownSignalChan(shutdownCh),
	    )
	    if err != nil {
	        log.Fatal(err)
	    }
	}

For a complete, runnable implementation see — in order:
  - examples/service/cmd/main.go
  - examples/service/internal/cli/cli.go
  - examples/service/internal/cli/bind.go
*/
package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/tecnickcom/gogen/pkg/logutil"
)

// Bootstrap is the function in charge of configuring the core components
// of an application and handling the lifecycle of its context.
func Bootstrap(bindFn BindFunc, opts ...Option) error {
	cfg := defaultConfig()

	for _, applyOpt := range opts {
		applyOpt(cfg)
	}

	err := cfg.validate()
	if err != nil {
		return err
	}

	// create application context
	ctx, cancel := context.WithCancel(cfg.context)
	defer cancel()

	m, err := cfg.createMetricsClientFunc()
	if err != nil {
		return fmt.Errorf("error creating application metric: %w", err)
	}

	if cfg.logConfig != nil {
		// metric hook to count logs by level
		cfg.logConfig.HookFn = func(level logutil.LogLevel, _ string) {
			m.IncLogLevelCounter(logutil.LevelName(level))
		}
	}

	l := cfg.createLoggerFunc()

	l.Debug("binding application components")

	err = bindFn(ctx, l, m)
	if err != nil {
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
func syncWaitGroupTimeout(wg *sync.WaitGroup, timeout time.Duration, l *slog.Logger) {
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
