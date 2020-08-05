package requestid

import (
	"context"
	"net/http"

	"github.com/nexmoinc/gosrvlib/pkg/httputil"
)

const (
	headerRequestID = "X-Request-Id"
)

type ctxKey struct{}

// FromContext retrieves the request ID from the context. If not found the defaultValue is returned instead
func FromContext(ctx context.Context, defaultValue string) string {
	if l, ok := ctx.Value(ctxKey{}).(string); ok {
		return l
	}
	return defaultValue
}

// WithRequestID builds a new context with the request ID to the given parent context.
// If the context already contains a request ID, it will not be overridden
func WithRequestID(ctx context.Context, id string) context.Context {
	if _, ok := ctx.Value(ctxKey{}).(string); ok {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, id)
}

// FromHTTPRequest retrieves the request ID from a http.Request. If not found the defaultValue is returned instead
func FromHTTPRequest(r *http.Request, defaultValue string) string {
	return httputil.HeaderOrDefault(r, headerRequestID, defaultValue)
}
