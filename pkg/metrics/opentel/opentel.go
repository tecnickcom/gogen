/*
Package opentel implements [github.com/tecnickcom/gogen/pkg/metrics.Client]
using OpenTelemetry for both metrics and tracing.

# Problem

Modern services need unified observability across HTTP boundaries, SQL calls,
and application-level error events. Implementing OpenTelemetry setup manually
in every service is repetitive and error-prone: providers, exporters,
propagators, resources, and shutdown sequencing must all be wired correctly.

# Solution

This package provides an OpenTelemetry-backed client with sensible defaults and
interface-compatible instrumentation hooks:

  - inbound HTTP handler instrumentation
  - outbound HTTP transport instrumentation
  - SQL open/instrumentation helpers with otelsql
  - log-level and error-taxonomy counters

At startup, [New] configures and registers global OpenTelemetry providers
(tracer, meter, propagator), creates default counters, and records shutdown
functions so callers can terminate exporters cleanly via [Client.Close] or
[Client.CloseCtx].

# Defaults and Configuration

By default, exporter selection is environment-driven:
  - Traces: OTLP/HTTP when OTEL_EXPORTER_OTLP_TRACES_ENDPOINT or
    OTEL_EXPORTER_OTLP_ENDPOINT is set; otherwise stdout exporter.
  - Metrics: OTLP/HTTP when OTEL_EXPORTER_OTLP_METRICS_ENDPOINT or
    OTEL_EXPORTER_OTLP_ENDPOINT is set; otherwise stdout exporter.

Resource attributes are resolved from explicit parameters and environment,
including OTEL_SERVICE_NAME, OTEL_SERVICE_VERSION,
OTEL_DEPLOYMENT_ENVIRONMENT_NAME, and OTEL_RESOURCE_ATTRIBUTES.

# Features

  - Unified metrics + tracing backend behind the shared metrics interface.
  - Automatic OpenTelemetry global provider setup.
  - HTTP server/client and SQL instrumentation primitives.
  - Context propagation support via configurable propagator.
  - Graceful exporter/provider shutdown via registered cleanup callbacks.
  - Optional trace ID extraction helpers for logging correlation.

# Benefits

This package reduces OpenTelemetry integration to one client while preserving
backend portability and keeping observability wiring consistent across services.
*/
package opentel
