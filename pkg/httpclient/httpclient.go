/*
Package httpclient provides a configurable outbound HTTP client with built-in
trace propagation and structured request/response logging.

# Problem

Service-to-service HTTP calls often require the same operational plumbing:
timeouts, trace ID forwarding, consistent logs, optional payload dumps in debug
mode, and redaction of sensitive data. Repeating this setup in each caller
creates inconsistent observability and duplicated boilerplate.

# Solution

This package wraps net/http client behavior in a small [Client] abstraction:
  - [New] builds a client from composable functional options.
  - [Client.Do] executes requests with trace/header enrichment and structured
    logging around request/response timing.

By default, the client uses a 1-minute timeout, the standard transport, and
the default trace ID header from the traceid package.

# Features

  - Configurable timeout, logger, component tag, log field prefix, and trace
    ID header name.
  - Automatic trace ID propagation: if a trace ID is missing in context, a new
    UUIDv7-based ID is generated and attached to context and request headers.
  - Structured outbound logging with request/response timestamps and duration.
  - Debug-level request/response dumps via net/http/httputil, with pluggable
    redaction before logs are written.
  - Transport extensibility through round-tripper wrapping and custom dial
    context hooks.

# Logging Behavior

At debug level, request and response dumps are logged (after redaction).
At non-debug levels, summary metadata is logged without payload dumps.
Errors are logged with the same trace context and timing fields.

Query strings are redacted before logging at every level, so secrets carried in
query parameters (for example api_key or token) are not written to logs. This
includes the error field of a failed request, whose embedded URL has its query
and userinfo redacted. Two things are not redacted: the request path (so avoid
placing secrets in the path, e.g. /reset/{token}), and errors produced by a
custom round-tripper that do not wrap a *url.Error (redact those in the
round-tripper).

Debug-level dumps are redacted by the configured function (see [WithRedactFn]),
which by default masks Authorization-style headers, cookie and other sensitive
key/value pairs, and card-like numbers. Arbitrary secret request headers (for
example a custom X-Api-Key) are not covered by the default; supply a stronger
redactor via [WithRedactFn] when such headers are sent.

Debug-level dumps buffer the request and response body in memory, bounded by a
configurable maximum (see [WithMaxDumpSize], default 1 MiB):

  - Request bodies larger than the cap, or of unknown length (streaming/chunked),
    have their headers dumped but their payload omitted; omitting unknown-length
    request bodies also avoids a deadlock that would otherwise occur when dumping
    a streaming request body.
  - Response bodies larger than the cap are omitted when the length is known; when
    the length is unknown (chunked/streaming), the body is truncated to the cap in
    the dump and marked as truncated, while the caller still receives the complete
    response body.

Because dumping an unknown-length response reads up to the cap before the log
entry is emitted (and before Do returns the body to the caller), debug-level
logging is not suitable for genuinely streaming endpoints (server-sent events,
long polling): enabling it can add up to WithMaxDumpSize worth of buffering
latency. Keep debug logging off for such endpoints, or lower the cap.

# Client Reuse

Each [New] builds a client with its own private transport and connection pool.
Create a client once and reuse it for the lifetime of the caller; constructing a
new client per request defeats connection reuse. Use
[Client.CloseIdleConnections] to release pooled connections on shutdown or
reconfiguration.

The default transport raises MaxIdleConnsPerHost above net/http's default of 2
so that per-host connection reuse is not throttled for the common case of many
concurrent calls to a few downstream hosts. Use [WithTLSClientConfig] for custom
CAs or client certificates, or [WithTransport] to supply a fully tuned transport
(pool sizes, proxy, HTTP/2) when the defaults do not fit.

# Benefits

httpclient standardizes outbound-call behavior across services, improving
traceability and diagnostics while keeping call-site code concise.
*/
package httpclient
