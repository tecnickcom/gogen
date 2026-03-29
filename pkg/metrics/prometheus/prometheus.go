/*
Package prometheus implements [github.com/tecnickcom/gogen/pkg/metrics.Client]
using the Prometheus client ecosystem.

# Problem

Service code usually needs consistent instrumentation for inbound HTTP traffic,
outbound HTTP calls, database pools, and application-level error taxonomy.
Without a shared implementation, teams duplicate metric definitions and produce
inconsistent labels, bucket strategies, and endpoint behavior.

# Solution

This package provides a Prometheus-backed client that registers a curated set
of default collectors and exposes wrappers for common instrumentation points:

  - inbound HTTP handlers
  - outbound HTTP round trippers
  - SQL database stats
  - application counters (log level and error code dimensions)

The implementation is based on:
  - github.com/prometheus/client_golang/prometheus
  - github.com/prometheus/client_golang/prometheus/promhttp
  - github.com/dlmiddlecote/sqlstats

# Default Metrics Scope

Default collectors include:
  - Go runtime and process metrics
  - HTTP server request count, in-flight gauge, duration histogram, request
    size histogram, and response size histogram
  - HTTP client request count, in-flight gauge, and duration histogram
  - error counters by level and by task/operation/code

# Benefits

The package gives developers production-ready, opinionated Prometheus
instrumentation in one place while keeping application code decoupled from
Prometheus-specific boilerplate.
*/
package prometheus
