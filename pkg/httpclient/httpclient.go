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

# Benefits

httpclient standardizes outbound-call behavior across services, improving
traceability and diagnostics while keeping call-site code concise.
*/
package httpclient
