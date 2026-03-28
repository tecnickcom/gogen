# gogen

[![GitHub Release](https://img.shields.io/github/v/release/tecnickcom/gogen)](https://github.com/tecnickcom/gogen/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/tecnickcom/gogen.svg)](https://pkg.go.dev/github.com/tecnickcom/gogen)
[![Coverage Status](https://coveralls.io/repos/github/tecnickcom/gogen/badge.svg?branch=main)](https://coveralls.io/github/tecnickcom/gogen?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/tecnickcom/gogen)](https://goreportcard.com/report/github.com/tecnickcom/gogen)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/11517/badge)](https://www.bestpractices.dev/projects/11517)

[![Donate via PayPal](https://img.shields.io/badge/donate-paypal-87ceeb.svg)](https://www.paypal.com/donate/?hosted_button_id=NZUEC5XS8MFBJ)
*Please consider supporting this project by making a donation via [PayPal](https://www.paypal.com/donate/?hosted_button_id=NZUEC5XS8MFBJ)*

![gogen logo](gogen_logo.png)

**This open-source project provides a collection of high-quality [Go](https://go.dev/) (Golang) [packages](#packages).**  

Have you ever struggled with stitching together the small but essential pieces of infrastructure for a Go service, such as configuration loading, retries, health checks, logging, metrics, AWS integration, validation, and cache handling, only to end up copying the same helper functions from project to project?

That's exactly the problem `gogen` solves.

### What makes it a good fit for Go teams

`gogen` is built for teams that want:

* A consistent set of high-quality utilities across services
* Reusable helper packages instead of ad hoc scripts
* A modular import model, not a framework lock-in
* A strong example-driven starting point via service
* A Makefile workflow for testing, building, and scaffolding

It also includes a generator path: `make project CONFIG=project.cfg` can scaffold a new web service, giving you a fast start with sane defaults.

If you need a solid set of reusable Go packages for web services, infrastructure glue, or AWS integrations, `gogen` is worth a look.

The source code documentation is available at:
[https://pkg.go.dev/github.com/tecnickcom/gogen/](https://pkg.go.dev/github.com/tecnickcom/gogen/)

-----------------------------------------------------------------

## TOC

* [Packages](#packages)
* [Quick Start](#quickstart)
* [Running all tests](#runtest)
* [Web-service project example](#example)

-----------------------------------------------------------------

<a name="packages"></a>

## Packages

**gogen** offers a comprehensive collection of well-tested Go packages.  
Each package adheres to common conventions and coding standards, making them easy to integrate into your projects.

* [awsopt](pkg/awsopt) - Utilities for configuring common AWS options with the aws-sdk-go-v2 library. [`aws`, `configuration`]
* [awssecretcache](pkg/awssecretcache) - Client for retrieving and caching secrets from AWS Secrets Manager. [`aws`, `secrets`, `caching`]
* [bootstrap](pkg/bootstrap) - Helpers for application bootstrap and initialization. [`bootstrap`, `initialization`]
* [config](pkg/config) - Utilities for configuration loading and management. [`configuration`]
* [countrycode](pkg/countrycode) - Functions for country code lookup and validation. [`geolocation`, `validation`]
* [countryphone](pkg/countryphone) - Phone number parsing and country association. [`phone`, `geolocation`, `parsing`]
* [decint](pkg/decint) - Helpers for parsing and formatting decimal integers. [`numeric`, `formatting`, `parsing`]
* [devlake](pkg/devlake) - Client for the DevLake Webhook API. [`webhook`, `api client`]
* [dnscache](pkg/dnscache) - DNS resolution with caching support. [`dns`, `caching`, `networking`]
* [encode](pkg/encode) - Utilities for data encoding and serialization. [`encoding`, `serialization`]
* [encrypt](pkg/encrypt) - Helpers for encryption and decryption. [`encryption`, `security`]
* [enumbitmap](pkg/enumbitmap) - Encode and decode slices of enumeration strings as integer bitmap values. [`enum`, `bitmap`, `encoding`]
* [enumcache](pkg/enumcache) - Caching for enumeration values with bitmap support. [`enum`, `caching`]
* [enumdb](pkg/enumdb) - Helpers for storing and retrieving enumeration sets in databases. [`enum`, `database`]
* [errutil](pkg/errutil) - Error utility functions, including error tracing. [`error handling`, `utilities`]
* [filter](pkg/filter) - Generic rule-based filtering for struct slices. [`filtering`, `collections`]
* [healthcheck](pkg/healthcheck) - Health check endpoints and logic. [`health`, `monitoring`]
* [httpclient](pkg/httpclient) - HTTP client with enhanced features. [`http`, `client`]
* [httpretrier](pkg/httpretrier) - HTTP request retry logic. [`http`, `retry`]
* [httpreverseproxy](pkg/httpreverseproxy) - HTTP reverse proxy implementation. [`http`, `reverse proxy`]
* [httpserver](pkg/httpserver) - HTTP server setup and management. [`http`, `server`]
* [httputil](pkg/httputil) - HTTP utility functions. [`http`, `utilities`]
  * [jsendx](pkg/httputil/jsendx) - Helpers for JSend-compliant responses. [`http`, `response formatting`]
* [ipify](pkg/ipify) - IP address lookup using the ipify service. [`ip lookup`, `networking`, `external service`]
* [jirasrv](pkg/jirasrv) - Client for Jira server APIs. [`api client`, `integration`]
* [jwt](pkg/jwt) - JSON Web Token creation and validation. [`jwt`, `authentication`, `security`]
* [kafka](pkg/kafka) - Kafka producer and consumer utilities. [`kafka`, `messaging`]
* [kafkacgo](pkg/kafkacgo) - Kafka integration using CGO bindings. [`kafka`, `messaging`, `cgo`]
* [logsrv](pkg/logsrv) - Default slog logger with zerolog handler. [`logging`, `slog`, `zerolog`]
* [logutil](pkg/logutil) - General log utilities for log/slog integration. [`logging`, `utilities`]
* [maputil](pkg/maputil) - Helpers for Go map manipulation. [`map utilities`, `collections`]
* [metrics](pkg/metrics) - Metrics collection and reporting. [`metrics`, `monitoring`]
  * [opentel](pkg/metrics/opentel) - OpenTelemetry metrics exporter (includes tracing). [`opentelemetry`, `metrics`, `tracing`]
  * [prometheus](pkg/metrics/prometheus) - Prometheus metrics exporter. [`prometheus`, `metrics`]
  * [statsd](pkg/metrics/statsd) - StatsD metrics exporter. [`statsd`, `metrics`]
* [mysqllock](pkg/mysqllock) - Distributed locking using MySQL. [`mysql`, `locking`, `distributed`]
* [numtrie](pkg/numtrie) - Trie data structure for numeric keys with partial matching. [`data structure`, `trie`]
* [paging](pkg/paging) - Helpers for data pagination. [`pagination`, `utilities`]
* [passwordhash](pkg/passwordhash) - Password hashing and verification. [`password hashing`, `security`]
* [passwordpwned](pkg/passwordpwned) - Password breach checking via HaveIBeenPwned. [`password breach`, `security`]
* [periodic](pkg/periodic) - Periodic task scheduling. [`scheduling`, `tasks`]
* [phonekeypad](pkg/phonekeypad) - Phone keypad mapping utilities. [`phone`, `mapping`, `utilities`]
* [profiling](pkg/profiling) - Application profiling tools. [`profiling`, `performance`]
* [random](pkg/random) - Utilities for random data generation, including UUID. [`random`, `utilities`]
* [redact](pkg/redact) - Data redaction helpers. [`redaction`, `privacy`]
* [redis](pkg/redis) - Redis client and utilities. [`redis`, `database`, `caching`]
* [retrier](pkg/retrier) - Retry logic for operations. [`retry`, `utilities`]
* [s3](pkg/s3) - Helpers for AWS S3 integration. [`aws`, `s3`]
* [sfcache](pkg/sfcache) - Simple, in-memory, thread-safe, fixed-size, single-flight cache for expensive lookups. [`caching`, `thread-safe`, `single-flight`]
* [slack](pkg/slack) - Client for sending messages via the Slack API Webhook. [`slack`, `webhook`, `messaging`]
* [sleuth](pkg/sleuth) - Client for the Sleuth.io API. [`api client`, `integration`]
* [sliceutil](pkg/sliceutil) - Utilities for slice manipulation. [`slice utilities`, `collections`]
* [sqlconn](pkg/sqlconn) - Helpers for SQL database connections. [`sql`, `database`]
* [sqltransaction](pkg/sqltransaction) - SQL transaction management. [`sql`, `transactions`]
* [sqlutil](pkg/sqlutil) - SQL utility functions. [`sql`, `utilities`]
* [sqlxtransaction](pkg/sqlxtransaction) - Helpers for SQLX transactions. [`sqlx`, `transactions`]
* [sqs](pkg/sqs) - Utilities for AWS SQS (Simple Queue Service) integration. [`aws`, `sqs`, `messaging`]
* [stringkey](pkg/stringkey) - Create unique hash keys from multiple strings. [`string keys`, `hashing`]
* [stringmetric](pkg/stringmetric) - String similarity and distance metrics. [`text similarity`, `metrics`]
* [strsplit](pkg/strsplit) - Utilities to split strings and Unicode text. [`string utilities`, `text`]
* [testutil](pkg/testutil) - Utilities for testing. [`testing`, `utilities`]
* [threadsafe](pkg/threadsafe) - Thread-safe data structures. [`thread-safe`, `concurrency`]
  * [tsmap](pkg/threadsafe/tsmap) - Thread-safe map implementation. [`thread-safe`, `map`]
  * [tsslice](pkg/threadsafe/tsslice) - Thread-safe slice implementation. [`thread-safe`, `slice`]
* [timeutil](pkg/timeutil) - Time and date utilities. [`time`, `date utilities`]
* [traceid](pkg/traceid) - Trace ID generation and management. [`tracing`, `ids`]
* [typeutil](pkg/typeutil) - Type conversion and utility functions. [`type conversion`, `utilities`]
* [validator](pkg/validator) - Data validation utilities. [`validation`, `utilities`]
* [valkey](pkg/valkey) - Wrapper client for interacting with valkey.io, an open-source in-memory data store. [`data store`, `client`]

-----------------------------------------------------------------

<a name="quickstart"></a>

## Developers' Quick Start

To get started quickly with this project, follow these steps:

1. Ensure you have the latest versions of Go and Python 3 installed (Python is required for additional tests).

2. Clone the repository:

    ```bash
    git clone https://github.com/tecnickcom/gogen.git
    ```

3. Navigate to the project directory:

    ```bash
    cd gogen
    ```

4. Install dependencies and run all tests:

    ```bash
    make x
    ```

You are now ready to start developing with gogen!

This project includes a *Makefile* that simplifies testing and building on Linux-compatible systems. All artifacts and reports generated by the *Makefile* are stored in the *target* folder.

Alternatively, you can build the project inside a [Docker](https://www.docker.com) container using:

```bash
make dbuild
```

This command uses the environment defined in `resources/docker/Dockerfile.dev`.

To view all available Makefile options, run:

```bash
make help
```

If you would like to contribute, please review the [CONTRIBUTING.md](https://github.com/tecnickcom/gogen/blob/main/CONTRIBUTING.md) guidelines.

-----------------------------------------------------------------

<a name="runtest"></a>

## Running all tests

Before committing your code, ensure it is properly formatted and passes all tests by running:

```bash
make x
```

Alternatively, you can build and test the project inside a [Docker](https://www.docker.com) container with:

```bash
make dbuild
```

-----------------------------------------------------------------

<a name="example"></a>

## Web-Service project example

Refer to the `examples/service` directory for a sample web service built using this library.

To create a new project based on the example and the settings defined in `project.cfg`, run:

```bash
make project CONFIG=project.cfg
```

-----------------------------------------------------------------
