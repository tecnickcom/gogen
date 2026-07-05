package httpserver

import "errors"

var (
	// ErrNilBinder is returned by New when a nil Binder is supplied.
	ErrNilBinder = errors.New("binder is required")

	// ErrNilRouteHandler is returned when a route is declared without a handler.
	ErrNilRouteHandler = errors.New("route handler is required")

	// ErrNilRouteMiddleware is returned when a route declares a nil middleware function.
	ErrNilRouteMiddleware = errors.New("route middleware function is required")

	// ErrInvalidRouteMethod is returned when a route has an empty HTTP method.
	ErrInvalidRouteMethod = errors.New("invalid route method")

	// ErrInvalidRoutePath is returned when a route path is empty or does not start with '/'.
	ErrInvalidRoutePath = errors.New("invalid route path")

	// ErrDuplicateRoute is returned when two routes share the same method and path.
	ErrDuplicateRoute = errors.New("duplicate route")

	// ErrRouteRegistration is returned when the underlying router rejects a route
	// (e.g. conflicting wildcard or malformed pattern).
	ErrRouteRegistration = errors.New("route registration failed")

	// ErrUnknownDefaultRoute is returned when an unknown default route identifier is enabled.
	ErrUnknownDefaultRoute = errors.New("unknown default route")

	// ErrInvalidTLSConfig is returned when a TLS configuration carries no
	// certificate material (no Certificates, GetCertificate, or GetConfigForClient).
	ErrInvalidTLSConfig = errors.New("invalid TLS configuration: no Certificates, GetCertificate, or GetConfigForClient set")
)
