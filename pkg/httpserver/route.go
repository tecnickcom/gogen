package httpserver

import (
	"net/http"
	"time"
)

// DisableTimeout is a sentinel [Route.Timeout] value that disables the request
// timeout for a single route, overriding any global timeout set with
// [WithRequestTimeout]. Use it for streaming or long-running endpoints (e.g.
// pprof profiles or server-sent events).
const DisableTimeout time.Duration = -1

// Route contains the HTTP route description.
type Route struct {
	// Method is the HTTP method (e.g.: GET, POST, PUT, DELETE, ...).
	// It must be uppercase: the router matches methods case-sensitively, so a
	// lowercase method would never match standard requests.
	Method string `json:"method"`

	// Path is the URL path.
	Path string `json:"path"`

	// Description is the description of this route that is displayed by the /index endpoint.
	Description string `json:"description"`

	// Handler is the handler function.
	Handler http.HandlerFunc `json:"-"`

	// Middleware is a set of middleware to apply to this route.
	Middleware []MiddlewareFn `json:"-"`

	// DisableLogger disable the default logger when set to true.
	DisableLogger bool `json:"-"`

	// Timeout time limit after which a request receives a 503 Service Unavailable.
	// A positive value overrides the common value set with WithRequestTimeout.
	// A negative value (see DisableTimeout) disables the timeout for this route.
	// Zero leaves the common value in effect.
	Timeout time.Duration `json:"-"`
}

// Index contains the list of routes attached to the current service.
type Index struct {
	// Routes is the list of routes attached to the current service.
	Routes []Route `json:"routes"`
}
