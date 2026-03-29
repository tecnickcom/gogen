# gogen

[![GitHub Release](https://img.shields.io/github/v/release/tecnickcom/gogen)](https://github.com/tecnickcom/gogen/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/tecnickcom/gogen.svg)](https://pkg.go.dev/github.com/tecnickcom/gogen)
[![Coverage Status](https://coveralls.io/repos/github/tecnickcom/gogen/badge.svg?branch=main)](https://coveralls.io/github/tecnickcom/gogen?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/tecnickcom/gogen)](https://goreportcard.com/report/github.com/tecnickcom/gogen)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/11517/badge)](https://www.bestpractices.dev/projects/11517)

[![Donate via PayPal](https://img.shields.io/badge/donate-paypal-87ceeb.svg)](https://www.paypal.com/donate/?hosted_button_id=NZUEC5XS8MFBJ)
Please consider supporting this project by making a donation via [PayPal](https://www.paypal.com/donate/?hosted_button_id=NZUEC5XS8MFBJ).

![gogen logo](gogen_logo.png)

**`gogen` is a production-oriented collection of modular, reusable Go packages for building services and infrastructure code.**

It solves a common problem in backend teams: repeatedly re-implementing the same foundational components (configuration loading, retries, health checks, logging, metrics, AWS integration, validation, caching, and more) across multiple repositories.

Instead of assembling and maintaining ad-hoc helpers per project, you can adopt tested packages with consistent patterns.

Source documentation: [pkg.go.dev/github.com/tecnickcom/gogen](https://pkg.go.dev/github.com/tecnickcom/gogen)

## Table of Contents

1. [Why gogen](#why-gogen)
2. [Feature Highlights](#feature-highlights)
3. [Benefits Summary](#benefits-summary)
4. [Package Catalog](#package-catalog)
5. [Developers Quick Start](#developers-quick-start)
6. [Running All Tests](#running-all-tests)
7. [Web Service Project Example](#web-service-project-example)
8. [Contributing](#contributing)

## Why gogen

`gogen` is a good fit for Go teams that want:

- A consistent utility layer across services
- Reusable packages rather than project-specific scripts
- A modular import model without framework lock-in
- A practical example service to accelerate onboarding
- A Makefile-driven workflow for test/build/scaffolding

It also includes a generator path:

```bash
make project CONFIG=project.cfg
```

This scaffolds a new web service from the provided configuration.

## Feature Highlights

- Broad package coverage for day-to-day service needs  
Why it matters: reduce dependency sprawl and avoid rewriting boilerplate utilities.

- Production-focused building blocks (HTTP, retries, observability, data stores, AWS)  
Why it matters: ship services faster with less glue code.

- Consistent conventions across packages  
Why it matters: easier code reviews, simpler maintenance, and predictable APIs.

- Testing-first repository culture  
Why it matters: safer refactoring and more reliable behavior over time.

- Built-in project scaffolding and runnable example service  
Why it matters: faster project bootstrap and clearer implementation reference.

## Benefits Summary

- Faster development cycles for new services
- Less duplicated utility code across repositories
- Better consistency in operational concerns (logging, metrics, health, tracing)
- Cleaner architecture through package-level composition
- Easier onboarding for engineers joining an existing Go platform

## Package Catalog

`gogen` offers a comprehensive set of well-tested packages.

- [awsopt](pkg/awsopt) - Utilities for configuring common AWS options with the aws-sdk-go-v2 library. `aws`, `configuration`
- [awssecretcache](pkg/awssecretcache) - Client for retrieving and caching secrets from AWS Secrets Manager. `aws`, `secrets`, `caching`
- [bootstrap](pkg/bootstrap) - Helpers for application bootstrap and initialization. `bootstrap`, `initialization`
- [config](pkg/config) - Utilities for configuration loading and management. `configuration`
- [countrycode](pkg/countrycode) - Functions for country code lookup and validation. `geolocation`, `validation`
- [countryphone](pkg/countryphone) - Phone number parsing and country association. `phone`, `geolocation`, `parsing`
- [decint](pkg/decint) - Helpers for parsing and formatting decimal integers. `numeric`, `formatting`, `parsing`
- [devlake](pkg/devlake) - Client for the DevLake Webhook API. `webhook`, `api client`
- [dnscache](pkg/dnscache) - DNS resolution with caching support. `dns`, `caching`, `networking`
- [encode](pkg/encode) - Utilities for data encoding and serialization. `encoding`, `serialization`
- [encrypt](pkg/encrypt) - Helpers for encryption and decryption. `encryption`, `security`
- [enumbitmap](pkg/enumbitmap) - Encode and decode slices of enumeration strings as integer bitmap values. `enum`, `bitmap`, `encoding`
- [enumcache](pkg/enumcache) - Caching for enumeration values with bitmap support. `enum`, `caching`
- [enumdb](pkg/enumdb) - Helpers for storing and retrieving enumeration sets in databases. `enum`, `database`
- [errutil](pkg/errutil) - Error utility functions, including error tracing. `error handling`, `utilities`
- [filter](pkg/filter) - Generic rule-based filtering for struct slices. `filtering`, `collections`
- [healthcheck](pkg/healthcheck) - Health check endpoints and logic. `health`, `monitoring`
- [httpclient](pkg/httpclient) - HTTP client with enhanced features. `http`, `client`
- [httpretrier](pkg/httpretrier) - HTTP request retry logic. `http`, `retry`
- [httpreverseproxy](pkg/httpreverseproxy) - HTTP reverse proxy implementation. `http`, `reverse proxy`
- [httpserver](pkg/httpserver) - HTTP server setup and management. `http`, `server`
- [httputil](pkg/httputil) - HTTP utility functions. `http`, `utilities`
- [jsendx](pkg/httputil/jsendx) - Helpers for JSend-compliant responses. `http`, `response formatting`
- [ipify](pkg/ipify) - IP address lookup using the ipify service. `ip lookup`, `networking`, `external service`
- [jirasrv](pkg/jirasrv) - Client for Jira server APIs. `api client`, `integration`
- [jwt](pkg/jwt) - JSON Web Token creation and validation. `jwt`, `authentication`, `security`
- [kafka](pkg/kafka) - Kafka producer and consumer utilities. `kafka`, `messaging`
- [kafkacgo](pkg/kafkacgo) - Kafka integration using CGO bindings. `kafka`, `messaging`, `cgo`
- [logsrv](pkg/logsrv) - Default slog logger with zerolog handler. `logging`, `slog`, `zerolog`
- [logutil](pkg/logutil) - General log utilities for log/slog integration. `logging`, `utilities`
- [maputil](pkg/maputil) - Helpers for Go map manipulation. `map utilities`, `collections`
- [metrics](pkg/metrics) - Metrics collection and reporting. `metrics`, `monitoring`
- [opentel](pkg/metrics/opentel) - OpenTelemetry metrics exporter (includes tracing). `opentelemetry`, `metrics`, `tracing`
- [prometheus](pkg/metrics/prometheus) - Prometheus metrics exporter. `prometheus`, `metrics`
- [statsd](pkg/metrics/statsd) - StatsD metrics exporter. `statsd`, `metrics`
- [mysqllock](pkg/mysqllock) - Distributed locking using MySQL. `mysql`, `locking`, `distributed`
- [numtrie](pkg/numtrie) - Trie data structure for numeric keys with partial matching. `data structure`, `trie`
- [paging](pkg/paging) - Helpers for data pagination. `pagination`, `utilities`
- [passwordhash](pkg/passwordhash) - Password hashing and verification. `password hashing`, `security`
- [passwordpwned](pkg/passwordpwned) - Password breach checking via HaveIBeenPwned. `password breach`, `security`
- [periodic](pkg/periodic) - Periodic task scheduling. `scheduling`, `tasks`
- [phonekeypad](pkg/phonekeypad) - Phone keypad mapping utilities. `phone`, `mapping`, `utilities`
- [profiling](pkg/profiling) - Application profiling tools. `profiling`, `performance`
- [random](pkg/random) - Utilities for random data generation, including UUID. `random`, `utilities`
- [redact](pkg/redact) - Data redaction helpers. `redaction`, `privacy`
- [redis](pkg/redis) - Redis client and utilities. `redis`, `database`, `caching`
- [retrier](pkg/retrier) - Retry logic for operations. `retry`, `utilities`
- [s3](pkg/s3) - Helpers for AWS S3 integration. `aws`, `s3`
- [sfcache](pkg/sfcache) - Simple in-memory, thread-safe, fixed-size, single-flight cache for expensive lookups. `caching`, `thread-safe`, `single-flight`
- [slack](pkg/slack) - Client for sending messages via the Slack API Webhook. `slack`, `webhook`, `messaging`
- [sleuth](pkg/sleuth) - Client for the Sleuth.io API. `api client`, `integration`
- [sliceutil](pkg/sliceutil) - Utilities for slice manipulation. `slice utilities`, `collections`
- [sqlconn](pkg/sqlconn) - Helpers for SQL database connections. `sql`, `database`
- [sqltransaction](pkg/sqltransaction) - SQL transaction management. `sql`, `transactions`
- [sqlutil](pkg/sqlutil) - SQL utility functions. `sql`, `utilities`
- [sqlxtransaction](pkg/sqlxtransaction) - Helpers for SQLX transactions. `sqlx`, `transactions`
- [sqs](pkg/sqs) - Utilities for AWS SQS (Simple Queue Service) integration. `aws`, `sqs`, `messaging`
- [stringkey](pkg/stringkey) - Create unique hash keys from multiple strings. `string keys`, `hashing`
- [stringmetric](pkg/stringmetric) - String similarity and distance metrics. `text similarity`, `metrics`
- [strsplit](pkg/strsplit) - Utilities to split strings and Unicode text. `string utilities`, `text`
- [testutil](pkg/testutil) - Utilities for testing. `testing`, `utilities`
- [threadsafe](pkg/threadsafe) - Thread-safe data structures. `thread-safe`, `concurrency`
- [tsmap](pkg/threadsafe/tsmap) - Thread-safe map implementation. `thread-safe`, `map`
- [tsslice](pkg/threadsafe/tsslice) - Thread-safe slice implementation. `thread-safe`, `slice`
- [timeutil](pkg/timeutil) - Time and date utilities. `time`, `date utilities`
- [traceid](pkg/traceid) - Trace ID generation and management. `tracing`, `ids`
- [typeutil](pkg/typeutil) - Type conversion and utility functions. `type conversion`, `utilities`
- [validator](pkg/validator) - Data validation utilities. `validation`, `utilities`
- [valkey](pkg/valkey) - Wrapper client for interacting with valkey.io, an open-source in-memory data store. `data store`, `client`

## Developers Quick Start

Requirements:

- Go (latest stable; repository is configured for Go 1.26)
- Python 3 (required for additional tests)

Clone and validate the repository:

```bash
git clone https://github.com/tecnickcom/gogen.git
cd gogen
make x
```

The `Makefile` provides a Linux-friendly workflow for build/test operations. Generated artifacts and reports are written to `target/`.

To run the same process in Docker:

```bash
make dbuild
```

This uses `resources/docker/Dockerfile.dev`.

List all available commands:

```bash
make help
```

## Running All Tests

Before committing, run:

```bash
make x
```

Or run tests/build inside Docker:

```bash
make dbuild
```

## Web Service Project Example

The directory `examples/service` contains a sample web service built with `gogen`.

To scaffold a new project using `project.cfg`:

```bash
make project CONFIG=project.cfg
```

## Contributing

Contributions are welcome. Please review [CONTRIBUTING.md](https://github.com/tecnickcom/gogen/blob/main/CONTRIBUTING.md) before opening a pull request.
