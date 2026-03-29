/*
Package main runs the gogenexample service.

# Overview

gogenexample is a production-oriented reference service built with gogen.
It demonstrates how to bootstrap a Go service with consistent configuration,
structured logging, metrics, health checks, and graceful shutdown.

The service solves a common problem for backend teams: starting from a clean,
maintainable foundation instead of rebuilding service plumbing for every new
project.

# Runtime Topology

By default, the process starts three HTTP servers:

- Monitoring server on :8071
- Private API server on :8072
- Public API server on :8073

The monitoring server exposes default operational endpoints (for example
index, ping, status, metrics, and profiling routes provided by gogen's
HTTP server package). The private and public APIs both expose a ping route
and the example /uid route.

The /uid route returns a UUIDv7 value, which is useful as an example of
application routes that can be attached to different exposure boundaries
(internal vs public).

# Configuration Model

Configuration is loaded through `gogen/pkg/config` and supports:

- Local configuration files
- Remote configuration providers (Consul, Etcd, env var payload)
- Environment variable overrides with the `GOGENEXAMPLE` prefix

Default ports and timeouts are defined in the CLI config package, and the
config schema supports enabling or disabling:

- The service handlers (`enabled`)
- Database connectivity (`db.enabled`)

When `db.enabled` is true, main and read connections are created, instrumented,
and included in health checks.

# Developer-Focused Features

  - Bootstrap lifecycle: startup and shutdown are coordinated with a shared
    wait group and shutdown signal channel.
  - Structured logging: configurable format and level with build metadata
    (`program`, `version`, `release`) attached to each record.
  - Observability-first HTTP stack: request instrumentation middleware,
    Prometheus metrics exposure, and standard failure handlers.
  - Health reporting: default status endpoint can be upgraded to dependency-
    aware health checks when optional components are enabled.
  - Trace propagation: outbound clients and servers use a shared trace header
    for request correlation.

# Why These Features Matter

  - Faster delivery: teams can implement business endpoints without spending
    early sprints rebuilding operational scaffolding.
  - Safer operations: built-in health, metrics, and graceful shutdown reduce
    deployment and incident risk.
  - Clear separation of concerns: public, private, and monitoring traffic
    are isolated by design.
  - Extensibility: service wiring in `internal/cli/bind.go` makes it easy to
    add dependencies, middleware, and handlers without changing core startup
    flow.

# Command-Line Flags

The root command supports common runtime overrides:

- `-c`, `--configDir`: add a configuration directory to the search list
- `-f`, `--logFormat`: set log format (`CONSOLE` or `JSON`)
- `-o`, `--logLevel`: set log level

The version subcommand prints the compiled program version.
*/
package main
