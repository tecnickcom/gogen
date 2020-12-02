//go:generate mockgen -package mocks -destination ../internal/mocks/httpserver_mocks.go . Router,Binder

// Package httpserver defines a default HTTP server with common routes.
package httpserver

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/nexmoinc/gosrvlib/pkg/httpserver/route"
	"github.com/nexmoinc/gosrvlib/pkg/httputil"
	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"github.com/nexmoinc/gosrvlib/pkg/metrics"
	"go.uber.org/zap"
)

// Router is the interface representing the router used by the HTTP http server
type Router interface {
	http.Handler
	Handler(method, path string, handler http.Handler)
}

// Binder is an interface to allow configuring the HTTP router
type Binder interface {
	BindHTTP(ctx context.Context) []route.Route
}

// NopBinder returns a simple no-operation binder
func NopBinder() Binder {
	return &nopBinder{}
}

type nopBinder struct{}

func (b *nopBinder) BindHTTP(_ context.Context) []route.Route { return nil }

// Start configures and start a new HTTP http server
func Start(ctx context.Context, binder Binder, opts ...Option) error {
	l := logging.WithComponent(ctx, "httpserver")

	cfg := defaultConfig()

	for _, applyOpt := range opts {
		if err := applyOpt(cfg); err != nil {
			return err
		}
	}

	if err := cfg.validate(); err != nil {
		return err
	}

	// mount status, metrics and pprof routes
	l.Debug("adding default routes")
	routes := newDefaultRoutes(cfg)

	// bind service routes
	l.Debug("adding service routes")
	customRoutes := binder.BindHTTP(ctx)

	// merge custom service routes with the default routes
	routes = append(routes, customRoutes...)

	for _, r := range routes {
		l.Debug("binding route", zap.String("path", r.Path))
		cfg.router.Handler(r.Method, r.Path, metrics.Handler(r.Path, r.Handler))
	}

	// attach route index if enabled
	if cfg.isIndexRouteEnabled() {
		l.Debug("enabling route index handler")
		cfg.router.Handler(http.MethodGet, indexPath, metrics.Handler(indexPath, cfg.indexHandlerFunc(routes)))
	}

	// wrap router with default middlewares
	return startServer(ctx, cfg)
}

func startServer(ctx context.Context, cfg *config) error {
	l := logging.FromContext(ctx)

	// create and start the http server
	s := &http.Server{
		Addr:         cfg.serverAddr,
		Handler:      requestInjectHandler(l, cfg.traceIDHeaderName, cfg.router),
		ReadTimeout:  cfg.serverReadTimeout,
		TLSConfig:    cfg.tlsConfig,
		WriteTimeout: cfg.serverWriteTimeout,
	}

	// start HTTP listener
	var ls net.Listener
	var err error
	if cfg.tlsConfig == nil {
		ls, err = net.Listen("tcp", cfg.serverAddr)
	} else {
		ls, err = tls.Listen("tcp", cfg.serverAddr, cfg.tlsConfig)
	}
	if err != nil {
		return err
	}
	l.Info("listening for HTTP requests", zap.String("addr", cfg.serverAddr))

	go func() {
		if err := s.Serve(ls); err != nil {
			l.Error("unexpected HTTP server failure", zap.Error(err))
		}
	}()

	go func() {
		<-ctx.Done()

		l.Debug("shutting down HTTP http server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.shutdownTimeout)
		defer cancel()

		_ = s.Shutdown(shutdownCtx)
		l.Debug("HTTP server shutdown")
	}()

	return nil
}

func defaultRouter() *httprouter.Router {
	r := httprouter.New()

	r.NotFound = metrics.Handler("404", func(w http.ResponseWriter, r *http.Request) {
		httputil.SendStatus(r.Context(), w, http.StatusNotFound)
	})

	r.MethodNotAllowed = metrics.Handler("405", func(w http.ResponseWriter, r *http.Request) {
		httputil.SendStatus(r.Context(), w, http.StatusMethodNotAllowed)
	})

	r.PanicHandler = func(w http.ResponseWriter, r *http.Request, p interface{}) {
		httputil.SendStatus(r.Context(), w, http.StatusInternalServerError)
	}

	return r
}

func defaultIndexHandler(routes []route.Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := &route.Index{Routes: routes}
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
