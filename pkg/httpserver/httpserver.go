/*
Package httpserver provides a configurable, production-oriented HTTP server
bootstrap for Go services.

# Problem

Starting a robust HTTP service usually requires repetitive infrastructure code:
listener setup, route registration, middleware chaining, panic/404/405
handling, request timeouts, TLS wiring, graceful shutdown, and optional
observability endpoints. Implementing this independently in each service leads
to inconsistency and duplicated maintenance effort.

# Solution

This package defines a reusable server assembly with:
  - option-driven configuration ([New] + functional [Option]s)
  - pluggable route binding via [Binder]
  - configurable default route set for operational endpoints
  - shared middleware application model
  - graceful shutdown orchestration via context and/or signal channel

Custom service routes are supplied by a [Binder], while default routes can be
enabled selectively (or all at once) through options.

# Default Operational Routes

When enabled, the built-in route set includes:
  - /ip: returns the service public IP (via ipify integration)
  - /metrics: returns metrics payload (501 by default unless replaced)
  - /ping: lightweight liveness endpoint
  - /pprof/*option: pprof profiling endpoints
  - /status: service health endpoint
  - / (index): generated route index

# Features

  - Graceful lifecycle control: non-blocking start, context-aware shutdown,
    configurable shutdown timeout, external wait-group and signal integration.
  - Router defaults: standardized not-found, method-not-allowed, and panic
    handlers with structured logging.
  - Middleware pipeline: common middleware (logger/timeout) plus global and
    per-route middleware composition.
  - Observability integration: trace-id propagation hooks, HTTP data redaction,
    and optional pprof/metrics/status routes.
  - Transport flexibility: plain TCP or TLS listener creation from cert/key
    material.
  - Safe defaults with extensibility: overridable handlers and server
    parameters for production customization.

# Benefits

httpserver reduces service bootstrap boilerplate, enforces consistent runtime
behavior across projects, and accelerates delivery of HTTP services with
operational best practices built in.

For a usage example, refer to examples/service/internal/cli/bind.go.
*/
package httpserver

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"

	"github.com/tecnickcom/gogen/pkg/httputil"
)

// Binder is an interface to allow configuring the HTTP router.
type Binder interface {
	// BindHTTP returns the routes.
	BindHTTP(ctx context.Context) []Route
}

// NopBinder returns no-operation binder that supplies no custom routes to router.
func NopBinder() Binder {
	return &nopBinder{}
}

// nopBinder is a no-operation binder implementation.
type nopBinder struct{}

// BindHTTP implements the Binder interface.
func (b *nopBinder) BindHTTP(_ context.Context) []Route { return nil }

// HTTPServer defines the HTTP Server object.
type HTTPServer struct {
	cfg          *config
	ctx          context.Context //nolint:containedctx
	httpServer   *http.Server
	listener     net.Listener
	startedMutex sync.Mutex
	started      bool
	shutdownOnce sync.Once
	shutdownDone chan struct{}
	monitorDone  chan struct{} // closed when the shutdown-monitor goroutine exits
}

// New constructs HTTP server with router, middleware, default operational routes, TLS, and graceful shutdown orchestration.
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
				IdleTimeout:       cfg.serverIdleTimeout,
				TLSConfig:         cfg.tlsConfig,
				WriteTimeout:      cfg.serverWriteTimeout,
			},
			listener:     listener,
			shutdownDone: make(chan struct{}),
			monitorDone:  make(chan struct{}),
		},
		nil
}

// StartServerCtx starts server in background goroutine with context-aware shutdown support.
func (h *HTTPServer) StartServerCtx(ctx context.Context) {
	// Register the running server with the wait group synchronously, before any
	// goroutine (including the shutdown one) can decrement it. The matching
	// decrement happens exactly once in Shutdown via sync.Once, and only for a
	// started server (Shutdown on a never-started server must not decrement).
	h.startedMutex.Lock()
	h.started = true
	h.cfg.shutdownWaitGroup.Add(1)
	h.startedMutex.Unlock()

	// wait for shutdown signal, direct Shutdown call, or context cancelation
	go func() { //nolint:gosec
		defer close(h.monitorDone)

		select {
		case <-h.cfg.shutdownSignalChan:
			h.cfg.logger.Debug("shutdown notification received")
		case <-h.shutdownDone:
			// Shutdown was already called directly: exit without leaking this goroutine.
			return
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

	h.cfg.logger.Info("listening for http requests")
}

// StartServer starts server in background using context from New().
func (h *HTTPServer) StartServer() {
	h.StartServerCtx(h.ctx)
}

// Shutdown gracefully shuts down server with timeout enforcement; wraps net/http Server.Shutdown().
// It is safe to call Shutdown any number of times and from multiple paths
// (e.g. manually and via the internal context/signal goroutine): the wait group
// is decremented exactly once, and only when the server was actually started.
func (h *HTTPServer) Shutdown(ctx context.Context) error {
	h.cfg.logger.Debug("shutting down http server")

	err := h.httpServer.Shutdown(ctx)

	h.shutdownOnce.Do(func() {
		// Unblock the internal shutdown-monitor goroutine (if any),
		// so a direct Shutdown call does not leak it.
		close(h.shutdownDone)

		h.startedMutex.Lock()
		defer h.startedMutex.Unlock()

		if h.started {
			h.cfg.shutdownWaitGroup.Add(-1)
			return
		}

		// http.Server.Shutdown only closes listeners registered by Serve,
		// so a never-started server must release its bound listener explicitly.
		cerr := h.listener.Close()
		if cerr != nil {
			h.cfg.logger.With(slog.Any("error", cerr)).Debug("failed closing the http server listener")
		}
	})

	h.cfg.logger.With(slog.Any("error", err)).Debug("http server shutdown complete")

	return err //nolint:wrapcheck
}

// serve starts serving HTTP requests.
func (h *HTTPServer) serve() {
	err := h.httpServer.Serve(h.listener)
	if errors.Is(err, http.ErrServerClosed) {
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
			Rnd:               cfg.rnd,
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
			Rnd:               cfg.rnd,
		}

		handler := ApplyMiddleware(args, cfg.indexHandlerFunc(routes), middleware...)

		cfg.router.Handler(args.Method, args.Path, handler)
	}
}
