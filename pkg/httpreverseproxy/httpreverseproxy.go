/*
Package httpreverseproxy provides a reusable reverse-proxy client built on top
of net/http/httputil.ReverseProxy.

# Problem

Service gateways and edge handlers often need to forward requests to upstream
services while preserving consistent forwarding semantics, transport setup, and
error observability. Rebuilding this plumbing for each proxy endpoint leads to
drift and duplicated maintenance.

# Solution

This package wraps ReverseProxy behind a focused [Client] API:
  - [New] configures proxy behavior from an upstream base address.
  - [Client.ForwardRequest] forwards incoming HTTP requests to the target.

When no custom rewrite function is provided, requests are rewritten to the
configured upstream URL and the wildcard `path` segment is forwarded as the
proxied path. Standard `X-Forwarded-*` headers are set automatically.

# Features

  - Rewrite-based upstream routing with sensible defaults.
  - Pluggable reverse proxy instance via [WithReverseProxy].
  - Pluggable HTTP transport client via [WithHTTPClient].
  - Structured proxy error logging with request metadata and response timing.
  - Default error handler returning HTTP 502 Bad Gateway on upstream failures.
  - Middleware-friendly forwarding entry point for integration with routers.

# Observability Behavior

The default error handler logs request method/path/query/URI, trace ID,
response code/status, request/response timing, and the underlying proxy error.
If request start time is present in context (via httputil request-time helpers),
response duration is computed from it.

# Benefits

httpreverseproxy reduces reverse-proxy boilerplate, keeps forwarding behavior
consistent across services, and improves operational visibility of upstream
failures.
*/
package httpreverseproxy
