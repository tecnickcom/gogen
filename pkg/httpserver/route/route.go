// Package route defines the endpoints for the HTTP server.
package route

import (
	"net/http"
)

// Middleware is a function that wraps a http.Handler.
type Middleware func(next http.Handler) http.Handler

// Route contains the HTTP route description.
type Route struct {
	// Method is the HTTP method (e.g.: GET, POST, PUT, DELETE, ...).
	Method string `json:"method"`

	// Path is the URL path.
	Path string `json:"path"`

	// Handler is the handler function.
	Handler http.HandlerFunc `json:"-"`

	// Description is the description of this route that is displayed by the /index endpoint.
	Description string `json:"description"`

	// Middlewares is a list of middlewares to apply to this route.
	Middlewares []Middleware `json:"-"`
}

// Index contains the list of routes attached to the current service.
type Index struct {
	// Routes is the list of routes attached to the current service.
	Routes []Route `json:"routes"`
}
