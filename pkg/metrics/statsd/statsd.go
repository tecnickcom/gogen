/*
Package statsd implements [github.com/tecnickcom/gogen/pkg/metrics.Client]
using the StatsD protocol.

# Problem

Many services want lightweight, low-latency metrics emission without exposing a
scrape endpoint in every process. StatsD-style push metrics fit this model, but
application code should not depend directly on backend-specific client APIs.

# Solution

This package provides a StatsD-backed metrics client compatible with the shared
metrics interface. It emits counters, gauges, and timers over UDP or TCP to a
StatsD daemon, which then forwards aggregated metrics to downstream systems
(such as Graphite-compatible storage).

Use [New] to create a client and configure it with options like [WithPrefix],
[WithNetwork], [WithAddress], and [WithFlushPeriod].

# Features

  - Inbound HTTP instrumentation: wraps handlers to emit request in/out
    counters, status-specific request counts, request/response sizes, and
    latency timers.
  - Outbound HTTP instrumentation: wraps [http.RoundTripper] to emit request
    in/out counters, response-status counts, and request latency.
  - Structured counter helpers: [Client.IncLogLevelCounter] and
    [Client.IncErrorCounter] encode operational and error-taxonomy events as
    StatsD counters.
  - Push-based transport: metrics are sent directly to StatsD over UDP
    (default) or TCP, suitable for environments where pull/scrape is not
    desired.
  - Shared interface compatibility: can be injected wherever the base metrics
    interface is used, enabling backend swapping without business-logic changes.

# Behavior Notes

StatsD is push-only in this implementation, so [Client.MetricsHandlerFunc]
returns HTTP 501 (Not Implemented). Database instrumentation via
[Client.InstrumentDB] is currently a no-op.

This package is based on github.com/tecnickcom/statsd.

# Benefits

The package offers a production-ready StatsD backend with minimal integration
friction, while preserving portability through the project-wide metrics
abstraction.
*/
package statsd
