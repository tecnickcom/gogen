package httpserver

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"
)

// Option is a type alias for a function that configures the HTTP httpServer instance
type Option func(*config) error

// WithEnableDefaultRoutes sets the default routes to be enabled on the server
func WithEnableDefaultRoutes(ids ...defaultRoute) Option {
	return func(cfg *config) error {
		cfg.defaultEnabledRoutes = ids
		return nil
	}
}

// WithEnableAllDefaultRoutes enables all default routes on the server
func WithEnableAllDefaultRoutes() Option {
	return func(cfg *config) error {
		cfg.defaultEnabledRoutes = allDefaultRoutes
		return nil
	}
}

// WithRouter replaces the default router used by the httpServer (mostly used for test purposes with a mock router)
func WithRouter(r Router) Option {
	return func(cfg *config) error {
		cfg.router = r
		return nil
	}
}

// WithMetricsHandlerFunc replaces the default metrics handler function
func WithMetricsHandlerFunc(handler http.HandlerFunc) Option {
	return func(cfg *config) error {
		cfg.metricsHandlerFunc = handler
		return nil
	}
}

// WithPingHandlerFunc replaces the default ping handler function
func WithPingHandlerFunc(handler http.HandlerFunc) Option {
	return func(cfg *config) error {
		cfg.pingHandlerFunc = handler
		return nil
	}
}

// WithPProfHandlerFunc replaces the default pprof handler function
func WithPProfHandlerFunc(handler http.HandlerFunc) Option {
	return func(cfg *config) error {
		cfg.pprofHandlerFunc = handler
		return nil
	}
}

// WithStatusHandlerFunc replaces the default status handler function
func WithStatusHandlerFunc(handler http.HandlerFunc) Option {
	return func(cfg *config) error {
		cfg.statusHandlerFunc = handler
		return nil
	}
}

// WithRoutesIndexHandlerFunc replaces the index handler
func WithRoutesIndexHandlerFunc(handler RouteIndexHandlerFunc) Option {
	return func(cfg *config) error {
		cfg.routeIndexHandlerFunc = handler
		return nil
	}
}

// WithServerAddr sets the address the httpServer will bind to
func WithServerAddr(addr string) Option {
	return func(cfg *config) error {
		cfg.serverAddr = addr
		return nil
	}
}

// WithServerReadTimeout sets the shutdown timeout
func WithServerReadTimeout(timeout time.Duration) Option {
	return func(cfg *config) error {
		cfg.serverReadTimeout = timeout
		return nil
	}
}

// WithServerWriteTimeout sets the shutdown timeout
func WithServerWriteTimeout(timeout time.Duration) Option {
	return func(cfg *config) error {
		cfg.serverWriteTimeout = timeout
		return nil
	}
}

// WithShutdownTimeout sets the shutdown timeout
func WithShutdownTimeout(timeout time.Duration) Option {
	return func(cfg *config) error {
		cfg.shutdownTimeout = timeout
		return nil
	}
}

// WithTLSCertData enable TLS with the given certificate and key data
func WithTLSCertData(pemCert, pemKey []byte) Option {
	return func(cfg *config) error {
		cert, err := tls.X509KeyPair(pemCert, pemKey)
		if err != nil {
			return fmt.Errorf("failed configuring TLS: %w", err)
		}
		cfg.tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		return nil
	}
}
