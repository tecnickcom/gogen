/*
Package traceid solves the distributed-tracing correlation problem at the
service boundary: capturing a request-scoped trace ID from an inbound HTTP
header, propagating it through the [context.Context] for the lifetime of the
request, and writing it back into outbound HTTP headers when calling downstream
services — all without coupling business logic to any particular tracing
framework.

# Problem

In distributed systems every request needs a stable identifier that survives
hops across services so that logs, metrics, and traces can be correlated after
the fact. The naive approach — passing the ID as an explicit function parameter
— pollutes every call site. Storing it ad-hoc under an untyped string key in the
context risks collisions with other packages. And extracting the ID from an HTTP
header requires consistent validation to prevent header-injection attacks.
This package encapsulates all three concerns in four small, focused functions.

# How It Works

The package uses an unexported struct type (ctxKey) as the context key,
guaranteeing no collision with any other package's context values.

  - [NewContext] stores the trace ID in the context. If an ID is already
    present the context is returned unchanged, making it safe to call at every
    layer without overwriting an upstream-supplied ID.
  - [FromContext] retrieves the stored ID, falling back to a caller-supplied
    default (typically [DefaultValue]) when none is set.
  - [FromHTTPRequestHeader] extracts the trace ID from an incoming HTTP request
    header and validates it against the pattern `^[0-9A-Za-z\-\_\.]{1,64}$`.
    Any value that does not match — including an empty or absent header — is
    replaced with the caller-supplied default, preventing header-injection.
  - [SetHTTPRequestHeaderFromContext] reads the ID from the context and writes
    it onto an outgoing [*http.Request] header, completing the propagation loop
    to downstream services.

The package also exports the conventional defaults [DefaultHeader] ("X-Request-ID"),
[DefaultValue] (""), and [DefaultLogKey] ("traceid") so all services in a system
can share consistent naming without hardcoding strings.

# Usage

At the inbound boundary (e.g. an HTTP middleware):

	id := traceid.FromHTTPRequestHeader(r, traceid.DefaultHeader, traceid.DefaultValue)
	ctx := traceid.NewContext(r.Context(), id)
	// pass ctx to all downstream handlers

Anywhere in the call chain — logging, metrics, business logic:

	id := traceid.FromContext(ctx, traceid.DefaultValue)
	logger.With(traceid.DefaultLogKey, id).Info("processing request")

At an outbound boundary (e.g. before calling a downstream service):

	traceid.SetHTTPRequestHeaderFromContext(ctx, outboundReq, traceid.DefaultHeader, traceid.DefaultValue)

This package is ideal for any Go HTTP service that participates in a distributed
system and needs lightweight, framework-agnostic trace ID propagation.
*/
package traceid

import (
	"context"
	"net/http"
	"regexp"
)

const (
	// DefaultHeader is the default header name for the trace ID.
	DefaultHeader = "X-Request-ID"

	// DefaultValue is the default trace ID value.
	DefaultValue = ""

	// DefaultLogKey is the default log field key for the Trace ID.
	DefaultLogKey = "traceid"
)

// regexPatternValidID is the regex pattern for a valid trace ID.
const regexPatternValidID = `^[0-9A-Za-z\-\_\.]{1,64}$`

// regexValidID is the compiled regex for a valid trace ID.
var regexValidID = regexp.MustCompile(regexPatternValidID)

// ctxKey is used to store the trace ID in the context.
type ctxKey struct{}

// NewContext stores the trace ID value in the context if not already present, making it safe to call at multiple layers.
func NewContext(ctx context.Context, id string) context.Context {
	if _, ok := ctx.Value(ctxKey{}).(string); ok {
		return ctx
	}

	return context.WithValue(ctx, ctxKey{}, id)
}

// FromContext retrieves the trace ID from context, returning the defaultValue if not found.
func FromContext(ctx context.Context, defaultValue string) string {
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		return v
	}

	return defaultValue
}

// SetHTTPRequestHeaderFromContext retrieves the trace ID from context and sets it onto the HTTP request header, validating it first; returns the ID that was set.
func SetHTTPRequestHeaderFromContext(ctx context.Context, r *http.Request, header, defaultValue string) string {
	id := FromContext(ctx, defaultValue)
	r.Header.Set(header, id)

	return id
}

// FromHTTPRequestHeader extracts the trace ID from the HTTP request header and validates it with regex pattern ^[0-9A-Za-z\-\_\.]{1,64}$; returns defaultValue if invalid or missing.
func FromHTTPRequestHeader(r *http.Request, header, defaultValue string) string {
	id := r.Header.Get(header)

	if !regexValidID.MatchString(id) {
		return defaultValue
	}

	return id
}
