package httpserver

import (
	"net/http"
)

// Router is deprecated.
// Deprecated: use *httprouter.Router instead.
type Router interface {
	http.Handler

	// Handler is an http.Handler wrapper.
	Handler(method, path string, handler http.Handler)
}

// InstrumentHandler is deprecated.
// Deprecated: Use instead WithMiddlewareFn.
type InstrumentHandler func(string, http.HandlerFunc) http.Handler

// WithInstrumentHandler is deprecated.
// Deprecated: Use WithMiddlewareFn instead.
func WithInstrumentHandler(handler InstrumentHandler) Option {
	return WithMiddlewareFn(
		func(args MiddlewareArgs, next http.Handler) http.Handler {
			return handler(args.Path, next.ServeHTTP)
		},
	)
}
