# gogen

[![Donate via PayPal](https://img.shields.io/badge/donate-paypal-87ceeb.svg)](https://www.paypal.com/donate/?hosted_button_id=NZUEC5XS8MFBJ)
*Please consider supporting this project by making a donation via [PayPal](https://www.paypal.com/donate/?hosted_button_id=NZUEC5XS8MFBJ)*

![gogen logo](gogen_logo.png)

**This open-source project provides a collection of high-quality [Go](https://go.dev/) (Golang) [packages](#packages).**  
Each package adheres to common conventions and can be imported individually into any project.  
These packages serve as a solid foundation for building fully-featured, production-ready web services.

You can generate a new web service by running `make project CONFIG=project.cfg`. The project's name, description, and other settings can be customized in the configuration file specified by the CONFIG parameter.

The package documentation is available at: [https://pkg.go.dev/github.com/tecnickcom/gogen/](https://pkg.go.dev/github.com/tecnickcom/gogen/)


[![GitHub Release](https://img.shields.io/github/v/release/tecnickcom/gogen)](https://github.com/tecnickcom/gogen/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/tecnickcom/gogen.svg)](https://pkg.go.dev/github.com/tecnickcom/gogen)
[![Coverage Status](https://coveralls.io/repos/github/tecnickcom/gogen/badge.svg?branch=main)](https://coveralls.io/github/tecnickcom/gogen?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/tecnickcom/gogen)](https://goreportcard.com/report/github.com/tecnickcom/gogen)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/11517/badge)](https://www.bestpractices.dev/projects/11517)

* **category**    Library
* **license**     [MIT](https://github.com/tecnickcom/gogen/blob/main/LICENSE)
* **link**        https://github.com/tecnickcom/gogen

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

* [awsopt](pkg/awsopt) - Utilities for configuring common AWS options with the aws-sdk-go-v2 library.
* [awssecretcache](pkg/awssecretcache) - Client for retrieving and caching secrets from AWS Secrets Manager.
* [bootstrap](pkg/bootstrap) - Helpers for application bootstrap and initialization.
* [config](pkg/config) - Utilities for configuration loading and management.
* [countrycode](pkg/countrycode) - Functions for country code lookup and validation.
* [countryphone](pkg/countryphone) - Phone number parsing and country association.
* [decint](pkg/decint) - Helpers for parsing and formatting decimal integers.
* [devlake](pkg/devlake) - Client for the DevLake Webhook API.
* [dnscache](pkg/dnscache) - DNS resolution with caching support.
* [encode](pkg/encode) - Utilities for data encoding and serialization.
* [encrypt](pkg/encrypt) - Helpers for encryption and decryption.
* [enumbitmap](pkg/enumbitmap) - Encode and decode slices of enumeration strings as integer bitmap values.
* [enumcache](pkg/enumcache) - Caching for enumeration values with bitmap support.
* [enumdb](pkg/enumdb) - Helpers for storing and retrieving enumeration sets in databases.
* [errutil](pkg/errutil) - Error utility functions, including error tracing.
* [filter](pkg/filter) - Generic rule-based filtering for struct slices.
* [healthcheck](pkg/healthcheck) - Health check endpoints and logic.
* [httpclient](pkg/httpclient) - HTTP client with enhanced features.
* [httpretrier](pkg/httpretrier) - HTTP request retry logic.
* [httpreverseproxy](pkg/httpreverseproxy) - HTTP reverse proxy implementation.
* [httpserver](pkg/httpserver) - HTTP server setup and management.
* [httputil](pkg/httputil) - HTTP utility functions.
  * [jsendx](pkg/httputil/jsendx) - Helpers for JSend-compliant responses.
* [ipify](pkg/ipify) - IP address lookup using the ipify service.
* [jirasrv](pkg/jirasrv) - Client for Jira server APIs.
* [jwt](pkg/jwt) - JSON Web Token creation and validation.
* [kafka](pkg/kafka) - Kafka producer and consumer utilities.
* [kafkacgo](pkg/kafkacgo) - Kafka integration using CGO bindings.
* [logsrv](pkg/logsrv) - Default slog logger with zerolog handler.
* [logutil](pkg/logutil) - General log utilities for log/slog integration.
* [maputil](pkg/maputil) - Helpers for Go map manipulation.
* [metrics](pkg/metrics) - Metrics collection and reporting.
  * [opentel](pkg/metrics/opentel) - OpenTelemetry metrics exporter (includes tracing).
  * [prometheus](pkg/metrics/prometheus) - Prometheus metrics exporter.
  * [statsd](pkg/metrics/statsd) - StatsD metrics exporter.
* [mysqllock](pkg/mysqllock) - Distributed locking using MySQL.
* [numtrie](pkg/numtrie) - Trie data structure for numeric keys with partial matching.
* [paging](pkg/paging) - Helpers for data pagination.
* [passwordhash](pkg/passwordhash) - Password hashing and verification.
* [passwordpwned](pkg/passwordpwned) - Password breach checking via HaveIBeenPwned.
* [periodic](pkg/periodic) - Periodic task scheduling.
* [phonekeypad](pkg/phonekeypad) - Phone keypad mapping utilities.
* [profiling](pkg/profiling) - Application profiling tools.
* [randkey](pkg/randkey) - Helpers for random key generation.
* [random](pkg/random) - Utilities for random data generation.
* [redact](pkg/redact) - Data redaction helpers.
* [redis](pkg/redis) - Redis client and utilities.
* [retrier](pkg/retrier) - Retry logic for operations.
* [s3](pkg/s3) - Helpers for AWS S3 integration.
* [sfcache](pkg/sfcache) - Simple, in-memory, thread-safe, fixed-size, single-flight cache for expensive lookups.
* [slack](pkg/slack) - Client for sending messages via the Slack API Webhook.
* [sleuth](pkg/sleuth) - Client for the Sleuth.io API.
* [sliceutil](pkg/sliceutil) - Utilities for slice manipulation.
* [sqlconn](pkg/sqlconn) - Helpers for SQL database connections.
* [sqltransaction](pkg/sqltransaction) - SQL transaction management.
* [sqlutil](pkg/sqlutil) - SQL utility functions.
* [sqlxtransaction](pkg/sqlxtransaction) - Helpers for SQLX transactions.
* [sqs](pkg/sqs) - Utilities for AWS SQS (Simple Queue Service) integration.
* [stringkey](pkg/stringkey) - Create unique hash keys from multiple strings.
* [stringmetric](pkg/stringmetric) - String similarity and distance metrics.
* [testutil](pkg/testutil) - Utilities for testing.
* [threadsafe](pkg/threadsafe) - Thread-safe data structures.
  * [tsmap](pkg/threadsafe/tsmap) - Thread-safe map implementation.
  * [tsslice](pkg/threadsafe/tsslice) - Thread-safe slice implementation.
* [timeutil](pkg/timeutil) - Time and date utilities.
* [traceid](pkg/traceid) - Trace ID generation and management.
* [typeutil](pkg/typeutil) - Type conversion and utility functions.
* [uidc](pkg/uidc) - Unique identifier generation.
* [validator](pkg/validator) - Data validation utilities.
* [valkey](pkg/valkey) - Wrapper client for interacting with valkey.io, an open-source in-memory data store.

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
