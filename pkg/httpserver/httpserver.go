/*
Package httpserver defines a default configurable HTTP server with common routes
and options.

Optional common routes are defined in the routes.go file. The routes include:
  - /ip: Returns the public IP address of the service instance.
  - /metrics: Returns Prometheus metrics (default and custom).
  - /ping: Pings the service to check if it is alive.
  - /pprof: Returns pprof profiling data for the selected profile.
  - /status: Checks and returns the health status of the service, including
    external services or components.

For a usage example, refer to the examples/service/internal/cli/bind.go file.
*/
package httpserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/tecnickcom/gogen/pkg/httputil"
)

// Binder is an interface to allow configuring the HTTP router.
type Binder interface {
	// BindHTTP returns the routes.
	BindHTTP(ctx context.Context) []Route
}

// NopBinder returns a simple no-operation binder.
func NopBinder() Binder {
	return &nopBinder{}
}

// nopBinder is a no-operation binder implementation.
type nopBinder struct{}

// BindHTTP implements the Binder interface.
func (b *nopBinder) BindHTTP(_ context.Context) []Route { return nil }

// HTTPServer defines the HTTP Server object.
type HTTPServer struct {
	cfg        *config
	ctx        context.Context //nolint:containedctx
	httpServer *http.Server
	listener   net.Listener
}

// New configures new HTTP server.
func New(ctx context.Context, binder Binder, opts ...Option) (*HTTPServer, error) {
	cfg := defaultConfig()

	for _, applyOpt := range opts {
		err := applyOpt(cfg)
		if err != nil {
			return nil, err
		}
	}

	cfg.logger = cfg.logger.With(
		slog.String("component", "httpserver"),
		slog.String("addr", cfg.serverAddr),
	)

	cfg.httpresp = httputil.NewHTTPResp(cfg.logger)

	cfg.setRouter(ctx)
	loadRoutes(ctx, binder, cfg)

	listener, err := netListener(ctx, cfg.serverAddr, cfg.tlsConfig)
	if err != nil {
		return nil, err
	}

	return &HTTPServer{
			cfg: cfg,
			ctx: ctx,
			httpServer: &http.Server{
				Addr:              cfg.serverAddr,
				Handler:           cfg.router,
				ReadHeaderTimeout: cfg.serverReadHeaderTimeout,
				ReadTimeout:       cfg.serverReadTimeout,
				TLSConfig:         cfg.tlsConfig,
				WriteTimeout:      cfg.serverWriteTimeout,
			},
			listener: listener,
		},
		nil
}

// StartServerCtx starts the current server and return without blocking.
// This ignore the context passed to the New() method.
func (h *HTTPServer) StartServerCtx(ctx context.Context) {
	// wait for shutdown signal or context cancelation
	go func() {
		select {
		case <-h.cfg.shutdownSignalChan:
			h.cfg.logger.Debug("shutdown notification received")
		case <-ctx.Done():
			h.cfg.logger.Warn("context canceled")
		}

		// The shutdown context is independent from the parent context.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), h.cfg.shutdownTimeout)
		defer cancel()

		_ = h.Shutdown(shutdownCtx) //nolint:contextcheck
	}()

	// start server
	go func() {
		h.serve()
	}()

	h.cfg.shutdownWaitGroup.Add(1)

	h.cfg.logger.Info("listening for http requests")
}

// StartServer starts the current server and return without blocking.
func (h *HTTPServer) StartServer() {
	h.StartServerCtx(h.ctx)
}

// Shutdown gracefully shuts down the server without interrupting any active connections.
// Wraps the standard net/http/Server_Shutdown method.
func (h *HTTPServer) Shutdown(ctx context.Context) error {
	h.cfg.logger.Debug("shutting down http server")

	err := h.httpServer.Shutdown(ctx)
	h.cfg.shutdownWaitGroup.Add(-1)

	h.cfg.logger.With(slog.Any("error", err)).Debug("http server shutdown complete")

	return err //nolint:wrapcheck
}

// serve starts serving HTTP requests.
func (h *HTTPServer) serve() {
	err := h.httpServer.Serve(h.listener)
	if err == http.ErrServerClosed {
		h.cfg.logger.Debug("closed http server")
		return
	}

	h.cfg.logger.With(slog.Any("error", err)).Error("unexpected http server failure")
}

// netListener creates a network listener for the given server address and TLS configuration.
func netListener(ctx context.Context, serverAddr string, tlsConfig *tls.Config) (net.Listener, error) {
	var (
		ls  net.Listener
		err error
	)

	if tlsConfig == nil {
		var lc net.ListenConfig

		ls, err = lc.Listen(ctx, "tcp", serverAddr)
	} else {
		ls, err = tls.Listen("tcp", serverAddr, tlsConfig)
	}

	if err != nil {
		return nil, fmt.Errorf("failed creating the http server address listener: %w", err)
	}

	return ls, nil
}

// loadRoutes loads and binds the routes to the HTTP server router.
func loadRoutes(ctx context.Context, binder Binder, cfg *config) {
	cfg.logger.Debug("loading default routes")

	routes := newDefaultRoutes(cfg)

	cfg.logger.Debug("loading service routes")

	customRoutes := binder.BindHTTP(ctx)

	routes = append(routes, customRoutes...)

	cfg.logger.Debug("applying routes")

	for _, r := range routes {
		cfg.logger.With(slog.String("path", r.Path)).Debug("binding route")

		// Add default and custom middleware functions
		middleware := cfg.commonMiddleware(r.DisableLogger, r.Timeout)
		middleware = append(middleware, r.Middleware...)

		args := MiddlewareArgs{
			Method:            r.Method,
			Path:              r.Path,
			Description:       r.Description,
			TraceIDHeaderName: cfg.traceIDHeaderName,
			RedactFunc:        cfg.redactFn,
			Logger:            cfg.logger,
		}

		handler := ApplyMiddleware(args, r.Handler, middleware...)

		cfg.router.Handler(r.Method, r.Path, handler)
	}

	// attach route index if enabled
	if cfg.isIndexRouteEnabled() {
		cfg.logger.Debug("enabling route index handler")

		_, disableLogger := cfg.disableDefaultRouteLogger[IndexRoute]
		middleware := cfg.commonMiddleware(disableLogger, 0)

		args := MiddlewareArgs{
			Method:            http.MethodGet,
			Path:              indexPath,
			Description:       "Index",
			TraceIDHeaderName: cfg.traceIDHeaderName,
			RedactFunc:        cfg.redactFn,
			Logger:            cfg.logger,
		}

		handler := ApplyMiddleware(args, cfg.indexHandlerFunc(routes), middleware...)

		cfg.router.Handler(args.Method, args.Path, handler)
	}
}
