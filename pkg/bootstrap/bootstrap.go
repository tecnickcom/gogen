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
    (see [WithShutdownSignalChan]) and the application context is canceled, so
    every registered dependent — whether keyed on the channel or on
    ctx.Done() — can start its own teardown.
 7. Bootstrap waits for all dependants to finish via a [sync.WaitGroup]
    (see [WithShutdownWaitGroup]), bounded by a configurable timeout
    (see [WithShutdownTimeout]) to prevent hanging indefinitely.
 8. The metrics client is closed so buffered measurements are flushed before
    the process exits.

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
	"github.com/tecnickcom/gogen/pkg/metrics"
)

// Bootstrap initializes core service infrastructure and manages process lifecycle.
//
// It solves the repeated startup/shutdown orchestration problem by centralizing
// context setup, logger and metrics creation, application binding, signal
// handling, and graceful termination in one call.
//
// The function applies options, validates configuration, invokes bindFn to wire
// dependants, waits for shutdown signals, broadcasts a shutdown event to all
// listeners, then waits for dependant completion up to the configured timeout.
//
// The main benefit is predictable service lifecycle behavior with less
// boilerplate and fewer shutdown edge-case bugs.
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
		// Work on a shallow copy so the caller-owned logutil.Config is never mutated.
		logCfg := *cfg.logConfig
		callerHookFn := logCfg.HookFn

		// metric hook to count logs by level, chained with any caller-installed hook
		logCfg.HookFn = func(level logutil.LogLevel, message string) {
			m.IncLogLevelCounter(logutil.LevelName(level))

			if callerHookFn != nil {
				callerHookFn(level, message)
			}
		}

		cfg.logConfig = &logCfg
	}

	l := cfg.createLoggerFunc()

	l.Debug("binding application components")

	err = bindFn(ctx, l, m)
	if err != nil {
		closeMetricsClient(m, l)

		return fmt.Errorf("application bootstrap error: %w", err)
	}

	l.Info("application started")

	done := make(chan struct{})

	// handle shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// stop relaying OS signals to quit once shutdown handling is done so the
	// registration does not outlive this Bootstrap call.
	defer signal.Stop(quit)

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

	// Send shutdown signal to all dependants (e.g. HTTP servers) by closing the
	// broadcast channel. The channel must be single-use: Bootstrap closes it
	// exactly once per call, so a caller-supplied channel (see
	// WithShutdownSignalChan) must not be shared across Bootstrap invocations or
	// closed elsewhere, otherwise this close panics.
	close(cfg.shutdownSignalChan)

	// Cancel the application context together with the shutdown broadcast, so
	// dependants whose teardown is keyed on ctx.Done() observe the cancellation
	// before Bootstrap starts waiting on the shutdown WaitGroup.
	cancel()

	// wait for graceful shutdown of dependants
	syncWaitGroupTimeout(cfg.shutdownWaitGroup, cfg.shutdownTimeout, l)

	// flush and release the metrics backend resources
	closeMetricsClient(m, l)

	l.Info("application stopped")

	return nil
}

// closeMetricsClient closes the metrics client, logging any close error.
//
// Closing flushes buffered measurements (e.g. statsd buffers, OTel readers) so
// the last metrics emitted before shutdown are not silently dropped.
func closeMetricsClient(m metrics.Client, l *slog.Logger) {
	err := m.Close()
	if err != nil {
		l.Error("error closing the metrics client", slog.Any("error", err))
	}
}

// syncWaitGroupTimeout waits for wg completion with an upper time bound.
//
// It prevents shutdown from hanging forever when a dependant goroutine fails to
// signal completion, and logs whether shutdown completed normally or timed out.
//
// NOTE: sync.WaitGroup offers no cancellable wait, so when the timeout fires
// the internal goroutine stays blocked in wg.Wait() until every dependant
// eventually calls Done. If a dependant never does, that goroutine leaks for
// the remaining process lifetime. This is an accepted trade-off: the typical
// caller exits right after Bootstrap returns, and the alternative would change
// the public WaitGroup-based dependant contract.
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
