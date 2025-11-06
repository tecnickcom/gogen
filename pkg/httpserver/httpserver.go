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
	"net"
	"net/http"

	"github.com/tecnickcom/gogen/pkg/httputil"
	"github.com/tecnickcom/gogen/pkg/logging"
	"go.uber.org/zap"
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
	logger     *zap.Logger
}

// Start configures and start a new HTTP server.
//
// Deprecated: Use New() and StartServerCtx() instead.
func Start(ctx context.Context, binder Binder, opts ...Option) error {
	h, err := New(ctx, binder, opts...)
	if err != nil {
		return err
	}

	h.StartServer()

	return nil
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

	logger := logging.WithComponent(ctx, "httpserver").With(
		zap.String("addr", cfg.serverAddr),
	)

	cfg.setRouter(ctx)
	loadRoutes(ctx, logger, binder, cfg)

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
			logger:   logger,
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
			h.logger.Debug("shutdown notification received")
		case <-ctx.Done():
			h.logger.Warn("context canceled")
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

	h.logger.Info("listening for http requests")
}

// StartServer starts the current server and return without blocking.
func (h *HTTPServer) StartServer() {
	h.StartServerCtx(h.ctx)
}

// Shutdown gracefully shuts down the server without interrupting any active connections.
// Wraps the standard net/http/Server_Shutdown method.
func (h *HTTPServer) Shutdown(ctx context.Context) error {
	h.logger.Debug("shutting down http server")

	err := h.httpServer.Shutdown(ctx)
	h.cfg.shutdownWaitGroup.Add(-1)

	h.logger.Debug("http server shutdown complete", zap.Error(err))

	return err //nolint:wrapcheck
}

// serve starts serving HTTP requests.
func (h *HTTPServer) serve() {
	err := h.httpServer.Serve(h.listener)
	if err == http.ErrServerClosed {
		h.logger.Debug("closed http server")
		return
	}

	h.logger.Error("unexpected http server failure", zap.Error(err))
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
func loadRoutes(ctx context.Context, l *zap.Logger, binder Binder, cfg *config) {
	l.Debug("loading default routes")

	routes := newDefaultRoutes(cfg)

	l.Debug("loading service routes")

	customRoutes := binder.BindHTTP(ctx)

	routes = append(routes, customRoutes...)

	l.Debug("applying routes")

	for _, r := range routes {
		l.Debug("binding route", zap.String("path", r.Path))

		// Add default and custom middleware functions
		middleware := cfg.commonMiddleware(r.DisableLogger, r.Timeout)
		middleware = append(middleware, r.Middleware...)

		args := MiddlewareArgs{
			Method:            r.Method,
			Path:              r.Path,
			Description:       r.Description,
			TraceIDHeaderName: cfg.traceIDHeaderName,
			RedactFunc:        cfg.redactFn,
			Logger:            l,
		}

		handler := ApplyMiddleware(args, r.Handler, middleware...)

		cfg.router.Handler(r.Method, r.Path, handler)
	}

	// attach route index if enabled
	if cfg.isIndexRouteEnabled() {
		l.Debug("enabling route index handler")

		_, disableLogger := cfg.disableDefaultRouteLogger[IndexRoute]
		middleware := cfg.commonMiddleware(disableLogger, 0)

		args := MiddlewareArgs{
			Method:            http.MethodGet,
			Path:              indexPath,
			Description:       "Index",
			TraceIDHeaderName: cfg.traceIDHeaderName,
			RedactFunc:        cfg.redactFn,
			Logger:            l,
		}

		handler := ApplyMiddleware(args, cfg.indexHandlerFunc(routes), middleware...)

		cfg.router.Handler(args.Method, args.Path, handler)
	}
}

// defaultIndexHandler returns the default index handler.
func defaultIndexHandler(routes []Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := &Index{Routes: routes}
		httputil.SendJSON(r.Context(), w, http.StatusOK, data)
	}
}

// defaultIPHandler returns the default /ip handler.
func defaultIPHandler(fn GetPublicIPFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusOK

		ip, err := fn(r.Context())
		if err != nil {
			status = http.StatusFailedDependency
		}

		httputil.SendText(r.Context(), w, status, ip)
	}
}

// defaultPingHandler returns the default /ping handler.
func defaultPingHandler(w http.ResponseWriter, r *http.Request) {
	httputil.SendStatus(r.Context(), w, http.StatusOK)
}

// defaultStatusHandler returns the default /status handler.
func defaultStatusHandler(w http.ResponseWriter, r *http.Request) {
	httputil.SendStatus(r.Context(), w, http.StatusOK)
}

// notImplementedHandler returns a 501 Not Implemented response.
func notImplementedHandler(w http.ResponseWriter, r *http.Request) {
	httputil.SendStatus(r.Context(), w, http.StatusNotImplemented)
}

// defaultNotFoundHandlerFunc returns the default 404 Not Found handler function.
func defaultNotFoundHandlerFunc(w http.ResponseWriter, r *http.Request) {
	httputil.SendStatus(r.Context(), w, http.StatusNotFound)
}

// defaultMethodNotAllowedHandlerFunc returns the default 405 Method Not Allowed handler function.
func defaultMethodNotAllowedHandlerFunc(w http.ResponseWriter, r *http.Request) {
	httputil.SendStatus(r.Context(), w, http.StatusMethodNotAllowed)
}

// defaultPanicHandlerFunc returns the default panic handler function.
func defaultPanicHandlerFunc(w http.ResponseWriter, r *http.Request) {
	httputil.SendStatus(r.Context(), w, http.StatusInternalServerError)
}
