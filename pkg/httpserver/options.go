package httpserver

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

// Option is a type alias for a function that configures the HTTP httpServer instance.
type Option func(*config) error

// WithRouter replaces the default router used by the httpServer (mostly used for test purposes with a mock router).
func WithRouter(r *httprouter.Router) Option {
	return func(cfg *config) error {
		cfg.router = r
		return nil
	}
}

// WithServerAddr sets the address the httpServer will bind to.
func WithServerAddr(addr string) Option {
	return func(cfg *config) error {
		cfg.serverAddr = addr
		return nil
	}
}

// WithServerReadHeaderTimeout sets the shutdown timeout.
func WithServerReadHeaderTimeout(timeout time.Duration) Option {
	return func(cfg *config) error {
		cfg.serverReadHeaderTimeout = timeout
		return nil
	}
}

// WithServerReadTimeout sets the shutdown timeout.
func WithServerReadTimeout(timeout time.Duration) Option {
	return func(cfg *config) error {
		cfg.serverReadTimeout = timeout
		return nil
	}
}

// WithServerWriteTimeout sets the shutdown timeout.
func WithServerWriteTimeout(timeout time.Duration) Option {
	return func(cfg *config) error {
		cfg.serverWriteTimeout = timeout
		return nil
	}
}

// WithShutdownTimeout sets the shutdown timeout.
func WithShutdownTimeout(timeout time.Duration) Option {
	return func(cfg *config) error {
		cfg.shutdownTimeout = timeout
		return nil
	}
}

// WithTLSCertData enable TLS with the given certificate and key data.
func WithTLSCertData(pemCert, pemKey []byte) Option {
	return func(cfg *config) error {
		cert, err := tls.X509KeyPair(pemCert, pemKey)
		if err != nil {
			return fmt.Errorf("failed configuring TLS: %w", err)
		}

		cfg.tlsConfig = &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
		}

		return nil
	}
}

// WithEnableDefaultRoutes sets the default routes to be enabled on the server.
func WithEnableDefaultRoutes(ids ...DefaultRoute) Option {
	return func(cfg *config) error {
		cfg.defaultEnabledRoutes = ids
		return nil
	}
}

// WithEnableAllDefaultRoutes enables all default routes on the server.
func WithEnableAllDefaultRoutes() Option {
	return func(cfg *config) error {
		cfg.defaultEnabledRoutes = allDefaultRoutes()
		return nil
	}
}

// WithIndexHandlerFunc replaces the index handler.
func WithIndexHandlerFunc(handler IndexHandlerFunc) Option {
	return func(cfg *config) error {
		cfg.indexHandlerFunc = handler
		return nil
	}
}

// WithIPHandlerFunc replaces the default ip handler function.
func WithIPHandlerFunc(handler http.HandlerFunc) Option {
	return func(cfg *config) error {
		cfg.ipHandlerFunc = handler
		return nil
	}
}

// WithMetricsHandlerFunc replaces the default metrics handler function.
func WithMetricsHandlerFunc(handler http.HandlerFunc) Option {
	return func(cfg *config) error {
		cfg.metricsHandlerFunc = handler
		return nil
	}
}

// WithPingHandlerFunc replaces the default ping handler function.
func WithPingHandlerFunc(handler http.HandlerFunc) Option {
	return func(cfg *config) error {
		cfg.pingHandlerFunc = handler
		return nil
	}
}

// WithPProfHandlerFunc replaces the default pprof handler function.
func WithPProfHandlerFunc(handler http.HandlerFunc) Option {
	return func(cfg *config) error {
		cfg.pprofHandlerFunc = handler
		return nil
	}
}

// WithStatusHandlerFunc replaces the default status handler function.
func WithStatusHandlerFunc(handler http.HandlerFunc) Option {
	return func(cfg *config) error {
		cfg.statusHandlerFunc = handler
		return nil
	}
}

// WithTraceIDHeaderName overrides the default trace id header name.
func WithTraceIDHeaderName(name string) Option {
	return func(cfg *config) error {
		cfg.traceIDHeaderName = name
		return nil
	}
}

// WithRedactFn set the function used to redact HTTP request and response dumps in the logs.
func WithRedactFn(fn RedactFn) Option {
	return func(cfg *config) error {
		cfg.redactFn = fn
		return nil
	}
}

// WithMiddlewareFn adds one or more middleware handler functions to all routes (endpoints).
// These middleware handlers are applied in the provided order after the default ones and before the custom route ones.
func WithMiddlewareFn(fn ...MiddlewareFn) Option {
	return func(cfg *config) error {
		cfg.middleware = append(cfg.middleware, fn...)
		return nil
	}
}

// WithNotFoundHandlerFunc http handler called when no matching route is found.
func WithNotFoundHandlerFunc(handler http.HandlerFunc) Option {
	return func(cfg *config) error {
		cfg.notFoundHandlerFunc = handler
		return nil
	}
}

// WithMethodNotAllowedHandlerFunc http handler called when a request cannot be routed.
func WithMethodNotAllowedHandlerFunc(handler http.HandlerFunc) Option {
	return func(cfg *config) error {
		cfg.methodNotAllowedHandlerFunc = handler
		return nil
	}
}

// WithPanicHandlerFunc http handler to handle panics recovered from http handlers.
func WithPanicHandlerFunc(handler http.HandlerFunc) Option {
	return func(cfg *config) error {
		cfg.panicHandlerFunc = handler
		return nil
	}
}

// WithoutRouteLogger disables the logger handler for all routes.
func WithoutRouteLogger() Option {
	return func(cfg *config) error {
		cfg.disableRouteLogger = true
		return nil
	}
}

// WithoutDefaultRouteLogger disables the logger handler for the specified default routes.
func WithoutDefaultRouteLogger(routes ...DefaultRoute) Option {
	return func(cfg *config) error {
		for _, route := range routes {
			cfg.disableDefaultRouteLogger[route] = true
		}

		return nil
	}
}
