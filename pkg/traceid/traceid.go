// Package traceid provide a simple mechanism to save/extract
// a Trace ID HTTP header to/from a context.Context and http.Request.
package traceid

import (
	"context"
	"net/http"
)

const (
	// DefaultHeader is the default header name for the trace ID
	DefaultHeader = "X-Request-ID"

	// DefaultValue is the default trace ID value.
	DefaultValue = ""
)

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
	if id == "" {
		return defaultValue
	}
	return id
}
