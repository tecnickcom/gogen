/*
Package traceid allows storing and retrieving a Trace ID value associated with a
context.Context and an HTTP request.

It provides functions to set and retrieve the trace ID from both the Context and
the HTTP request headers.

The Trace ID is typically used in distributed systems to track requests as they
propagate through different services. It can be used for debugging, performance
monitoring, and troubleshooting purposes.

The Trace ID is expected to be a string that follows the regex pattern
"^[0-9A-Za-z\-\_\.]{1,64}$". If the Trace ID does not match this pattern, the
default value is used instead.
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

const regexPatternValidID = `^[0-9A-Za-z\-\_\.]{1,64}$`

var regexValidID = regexp.MustCompile(regexPatternValidID)

// ctxKey is used to store the trace ID in the context.
type ctxKey struct{}

// NewContext stores the trace ID value in the context if not already present.
func NewContext(ctx context.Context, id string) context.Context {
	if _, ok := ctx.Value(ctxKey{}).(string); ok {
		return ctx
	}

	return context.WithValue(ctx, ctxKey{}, id)
}

// FromContext returns the trace ID associated with the context.
// If no trace ID is associated, then the default value returned.
func FromContext(ctx context.Context, defaultValue string) string {
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		return v
	}

	return defaultValue
}

// SetHTTPRequestHeaderFromContext set the trace ID HTTP Request Header with the value retrieved from the context.
// If the traceid is not found in the context, then the default value is set.
// Returns the set ID.
func SetHTTPRequestHeaderFromContext(ctx context.Context, r *http.Request, header, defaultValue string) string {
	id := FromContext(ctx, defaultValue)
	r.Header.Set(header, id)

	return id
}

// FromHTTPRequestHeader retrieves the trace ID from an HTTP Request.
// If not found the default value is returned instead.
func FromHTTPRequestHeader(r *http.Request, header, defaultValue string) string {
	id := r.Header.Get(header)

	if !regexValidID.MatchString(id) {
		return defaultValue
	}

	return id
}
