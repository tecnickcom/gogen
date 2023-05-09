// Package httpserver defines a default HTTP server with common routes.
package httpserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	"github.com/Vonage/gosrvlib/pkg/httputil"
	"github.com/Vonage/gosrvlib/pkg/logging"
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

type nopBinder struct{}

func (b *nopBinder) BindHTTP(_ context.Context) []Route { return nil }

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

// Start configures and start a new HTTP http server.
func Start(ctx context.Context, binder Binder, opts ...Option) error {
	l := logging.WithComponent(ctx, "httpserver")

	cfg := defaultConfig()

	for _, applyOpt := range opts {
		if err := applyOpt(cfg); err != nil {
			return err
		}
	}

	cfg.setRouter(ctx)

	if err := cfg.validate(); err != nil {
		return err
	}

	loadRoutes(ctx, l, binder, cfg)

	// wrap router with default middlewares
	return startServer(ctx, cfg)
}

func startServer(ctx context.Context, cfg *config) error {
	l := logging.FromContext(ctx)

	// create and start the http server
	s := &http.Server{
		Addr:              cfg.serverAddr,
		Handler:           cfg.router,
		ReadHeaderTimeout: cfg.serverReadHeaderTimeout,
		ReadTimeout:       cfg.serverReadTimeout,
		TLSConfig:         cfg.tlsConfig,
		WriteTimeout:      cfg.serverWriteTimeout,
	}

	// start HTTP listener
	var (
		ls  net.Listener
		err error
	)

	if cfg.tlsConfig == nil {
		ls, err = net.Listen("tcp", cfg.serverAddr)
	} else {
		ls, err = tls.Listen("tcp", cfg.serverAddr, cfg.tlsConfig)
	}

	if err != nil {
		return fmt.Errorf("failed creting the address listener: %w", err)
	}

	logAddr := zap.String("addr", cfg.serverAddr)
	l.Info("listening for HTTP requests", logAddr)

	go func() {
		serve(s, ls, l, logAddr)
	}()

	go func() {
		<-ctx.Done()

		l.Debug("shutting down HTTP http server", logAddr)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.shutdownTimeout)
		defer cancel()

		_ = s.Shutdown(shutdownCtx)

		cfg.shutdownWaitGroup.Add(-1)

		l.Debug("HTTP server shutdown", logAddr)
	}()

	cfg.shutdownWaitGroup.Add(1)

	return nil
}

func serve(s *http.Server, ls net.Listener, l *zap.Logger, logAddr zap.Field) {
	err := s.Serve(ls)
	if err == http.ErrServerClosed {
		l.Debug("closed HTTP http server", logAddr)
		return
	}

	l.Error("unexpected HTTP server failure", logAddr, zap.Error(err))
}

func defaultIndexHandler(routes []Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := &Index{Routes: routes}
		httputil.SendJSON(r.Context(), w, http.StatusOK, data)
	}
}

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

func defaultPingHandler(w http.ResponseWriter, r *http.Request) {
	httputil.SendStatus(r.Context(), w, http.StatusOK)
}

func defaultStatusHandler(w http.ResponseWriter, r *http.Request) {
	httputil.SendStatus(r.Context(), w, http.StatusOK)
}

func notImplementedHandler(w http.ResponseWriter, r *http.Request) {
	httputil.SendStatus(r.Context(), w, http.StatusNotImplemented)
}

func defaultNotFoundHandlerFunc(w http.ResponseWriter, r *http.Request) {
	httputil.SendStatus(r.Context(), w, http.StatusNotFound)
}

func defaultMethodNotAllowedHandlerFunc(w http.ResponseWriter, r *http.Request) {
	httputil.SendStatus(r.Context(), w, http.StatusMethodNotAllowed)
}

func defaultPanicHandlerFunc(w http.ResponseWriter, r *http.Request) {
	httputil.SendStatus(r.Context(), w, http.StatusInternalServerError)
}
