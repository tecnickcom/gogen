// Package traceid provide a simple mechanism to save/extract
// a Trace ID HTTP header to/from a context.Context and http.Request.
package traceid

import (
	"context"
	"net/http"

	"github.com/nexmoinc/gosrvlib/pkg/httputil"
)

const (
	// DefaultKey is the default header name for the trace ID
	DefaultKey = "X-Request-ID"
	// DefaultValue is the default trace ID value.
	DefaultValue = ""
)

// ctxKey is the context key used to store the trace ID value
type ctxKey struct{}

// ToContext stores the trace ID value in the context if not already present.
func ToContext(ctx context.Context, id string) context.Context {
	if _, ok := ctx.Value(ctxKey{}).(string); ok {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, id)
}

// FromContext returns the trace ID associated with the context.
// If no trace ID is associated, then the default value returned.
func FromContext(ctx context.Context, def string) string {
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		return v
	}
	return def
}

// ToHTTPRequest set the trace ID HTTP Request Header with the value retrieved from the context.
// If the traceid is not found in the context, then the default value is set.
// Returns true when the default value is used.
func ToHTTPRequest(ctx context.Context, r *http.Request, key, def string) bool {
	id := FromContext(ctx, def)
	r.Header.Set(key, id)
	return id == def
}

// FromHTTPRequest retrieves the trace ID from an HTTP Request.
// If not found the defaultValue is returned instead.
func FromHTTPRequest(r *http.Request, key, def string) string {
	return httputil.HeaderOrDefault(r, key, def)
}
