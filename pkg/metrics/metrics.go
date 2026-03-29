/*
Package metrics defines a backend-agnostic instrumentation contract for Go
services.

# Problem

Application code often needs to instrument SQL connections, HTTP handlers,
outgoing HTTP clients, and domain-level error counters. If business code calls
vendor-specific APIs directly (Prometheus, StatsD, OpenTelemetry, etc.),
switching backend or running tests without a metrics stack becomes difficult.

# Solution

This package provides the [Client] interface as a stable abstraction over
common instrumentation points used across services:

  - SQL opening and DB instrumentation
  - inbound HTTP handler instrumentation
  - outbound HTTP round-tripper instrumentation
  - metrics endpoint handler
  - application-level counters (log levels and error taxonomy)

The [Default] implementation is intentionally minimal and safe:

  - it behaves as a no-op for instrumentation methods,
  - it still returns a valid SQL connection from [Default.SqlOpen], and
  - it exposes a lightweight health-style metrics handler that returns "OK".

This makes instrumentation optional and allows applications to run without a
configured metrics backend while keeping a consistent API.

# Backends

Concrete implementations are available in sibling packages:
  - github.com/tecnickcom/gogen/pkg/metrics/statsd
  - github.com/tecnickcom/gogen/pkg/metrics/prometheus
  - github.com/tecnickcom/gogen/pkg/metrics/opentel

# Benefits

Code can depend on one small interface and remain portable across metrics
systems, easier to test, and simpler to evolve as observability requirements
change.
*/
package metrics

import (
	"database/sql"
	"net/http"
)

// Client defines the instrumentation surface consumed by the rest of the
// application.
//
// Implementations may export metrics to Prometheus, StatsD, OpenTelemetry, or
// any custom backend, while callers stay decoupled from backend-specific APIs.
type Client interface {
	// SqlOpen wraps sql.Open and may add driver-level instrumentation.
	//
	// Implementations that do not support SQL instrumentation should still
	// return a valid *sql.DB using the provided driverName and dsn.
	SqlOpen(driverName, dsn string) (*sql.DB, error)

	// InstrumentDB attaches instrumentation to an existing DB instance.
	//
	// dbName is the logical name used as a metrics label.
	InstrumentDB(dbName string, db *sql.DB) error

	// InstrumentHandler wraps an inbound HTTP handler to collect request metrics
	// (for example latency, status code, and request counts).
	InstrumentHandler(path string, handler http.HandlerFunc) http.Handler

	// InstrumentRoundTripper wraps an outbound HTTP transport to collect client
	// request metrics.
	InstrumentRoundTripper(next http.RoundTripper) http.RoundTripper

	// MetricsHandlerFunc returns the HTTP handler for the metrics endpoint.
	MetricsHandlerFunc() http.HandlerFunc

	// IncLogLevelCounter increments a counter by log severity level.
	IncLogLevelCounter(level string)

	// IncErrorCounter increments an application error counter partitioned by
	// task, operation, and code labels.
	IncErrorCounter(task, operation, code string)

	// Close flushes or tears down backend resources.
	//
	// It should be safe to call during service shutdown.
	Close() error
}

// Default is a no-op implementation of [Client].
//
// It is useful as a safe fallback when no metrics backend is configured,
// allowing instrumentation calls to remain in place without side effects.
type Default struct{}

// SqlOpen delegates to sql.Open without additional instrumentation.
func (c *Default) SqlOpen(driverName, dsn string) (*sql.DB, error) {
	return sql.Open(driverName, dsn) //nolint:wrapcheck
}

// InstrumentDB is a no-op and returns nil.
func (c *Default) InstrumentDB(_ string, _ *sql.DB) error {
	return nil
}

// InstrumentHandler returns handler unchanged.
func (c *Default) InstrumentHandler(_ string, handler http.HandlerFunc) http.Handler {
	return handler
}

// InstrumentRoundTripper returns next unchanged.
func (c *Default) InstrumentRoundTripper(next http.RoundTripper) http.RoundTripper {
	return next
}

// MetricsHandlerFunc returns a minimal handler that responds with "OK".
func (c *Default) MetricsHandlerFunc() http.HandlerFunc {
	// Returns "OK" by default.
	return func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`OK`)) }
}

// IncLogLevelCounter is a no-op.
func (c *Default) IncLogLevelCounter(_ string) {
	// Do nothing.
	_ = 0
}

// IncErrorCounter is a no-op.
func (c *Default) IncErrorCounter(_, _, _ string) {
	// Do nothing.
	_ = 0
}

// Close is a no-op and returns nil.
func (c *Default) Close() error {
	return nil
}
