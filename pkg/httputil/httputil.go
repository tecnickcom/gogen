/*
Package httputil provides reusable HTTP request/response primitives for Go
services built on top of net/http.

# Problem

HTTP handlers frequently repeat the same infrastructure code: setting JSON/auth
headers, parsing query defaults, extracting router path params, tracking request
timing, writing consistent response payloads, and instrumenting response
writers. Duplicating this logic across handlers increases inconsistency and
maintenance overhead.

# Solution

This package centralizes that boilerplate into focused helpers:
  - request helpers: header decorators, path/query/header default extraction,
    request-time context utilities
  - response helpers: text/JSON/XML responses with no-cache and nosniff headers,
    full-buffer encoding with a 500 fallback on marshal failure, and structured
    response logging via [HTTPResp]
  - response-writer wrapper: status/size capture and optional tee support for
    middleware instrumentation ([ResponseWriterWrapper])
  - URL composition helper for link building ([Link])

# Highlights

  - Header helpers for JSON, Basic Auth, and Bearer tokens.
  - Safe query/header parsing with typed defaults for string/int/uint.
  - Context-based request start-time propagation and retrieval.
  - Uniform response writing with MIME constants and JSend-style status
    projection and round-trip parsing ([StatusSuccess], [StatusFail],
    [StatusError], [Status.UnmarshalJSON], [ErrInvalidStatus]).
  - Structured response logs containing code, duration, timestamp, trace ID, and
    payload metadata, logged by class (2xx debug, 4xx warn, 5xx error).
  - Middleware-friendly response writer proxy exposing written status and byte
    size.

# Benefits

httputil reduces repetitive handler scaffolding, improves request/response
consistency, and makes HTTP middleware stacks easier to observe and maintain.
*/
package httputil
