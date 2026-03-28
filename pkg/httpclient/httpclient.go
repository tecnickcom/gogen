/*
Package httpclient provides a configurable, instrumented HTTP client for service
communication.

It solves the problem of building request pipelines with consistent tracing,
timeouts, retries, and logging behavior without copying the same client setup
logic across services.

The package includes support for trace ID headers and optional request/response
logging, including redaction of sensitive data.

Top features:
- configurable HTTP client construction
- trace ID propagation for distributed tracing
- request/response dumping with redaction support
- reusable transport and middleware-friendly design

Benefits:
- reduce HTTP client setup boilerplate
- make outbound service calls easier to observe
- keep logging and tracing consistent across services
*/
package httpclient
