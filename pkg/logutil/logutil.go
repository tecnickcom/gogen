/*
Package logutil provides configuration-driven logging utilities built around
Go's standard log/slog package.

# Problem

Applications often need to initialize logging consistently across environments
(JSON in production, human-readable output in development), map string-based
configuration to valid log levels/formats, attach common structured fields,
bridge legacy log.Logger output, and optionally run side effects when messages
are emitted. Rewriting this setup logic in every service leads to boilerplate
and inconsistent behavior.

# Solution

This package wraps slog with a small, composable configuration model:
  - [Config] and [Option] build loggers from strongly typed settings.
  - [Config.SlogLogger], [Config.SlogHandler], and
    [Config.SlogDefaultLogger] construct handler/logger instances.
  - [ParseLevel]/[ParseFormat] and [ValidLevel]/[ValidFormat] convert and
    validate runtime configuration values safely.
  - [NewSlogHookHandler] allows interception of log messages via [HookFunc].
  - [NewSlogWriter] and [NewLogFromSlog] bridge standard log.Logger output
    into slog.

# Features

  - Multi-format output: JSON, text console, or discard mode.
  - Extended severity model: supports syslog-style levels (emergency through
    debug) plus trace.
  - Centralized common attributes: attach service-wide structured fields once.
  - Hook integration: execute custom side effects for each log record
    (for example, metrics counters or external notifications).
  - Legacy compatibility: route existing standard library logger output into
    structured slog pipelines without rewriting call sites.
  - Zero external runtime dependency: implemented with the Go standard library.

# Benefits

logutil standardizes logger initialization and behavior across codebases,
reducing setup boilerplate while improving consistency, observability, and
operational safety.
*/
package logutil
