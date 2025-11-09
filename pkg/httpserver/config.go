package httpserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/tecnickcom/gogen/pkg/httputil"
	"github.com/tecnickcom/gogen/pkg/ipify"
	"github.com/tecnickcom/gogen/pkg/profiling"
	"github.com/tecnickcom/gogen/pkg/redact"
	"github.com/tecnickcom/gogen/pkg/traceid"
)

// timeoutMessage is the message used for timeout responses.
const timeoutMessage = "TIMEOUT"

// RedactFn is an alias for a redact function.
type RedactFn func(s string) string

// IndexHandlerFunc is a type alias for the route index function.
type IndexHandlerFunc func([]Route) http.HandlerFunc

// GetPublicIPFunc is a type alias for function to get public IP of the service.
type GetPublicIPFunc func(context.Context) (string, error)

// GetPublicIPDefaultFunc returns the GetPublicIP function for a default ipify client.
func GetPublicIPDefaultFunc() GetPublicIPFunc {
	c, _ := ipify.New() // no errors are returned with default values
	return c.GetPublicIP
}

// config contains the configuration for the HTTP server.
type config struct {
	router                      *httprouter.Router
	serverAddr                  string
	traceIDHeaderName           string
	requestTimeout              time.Duration
	serverReadHeaderTimeout     time.Duration
	serverReadTimeout           time.Duration
	serverWriteTimeout          time.Duration
	shutdownTimeout             time.Duration
	tlsConfig                   *tls.Config
	defaultEnabledRoutes        []DefaultRoute
	indexHandlerFunc            IndexHandlerFunc
	ipHandlerFunc               http.HandlerFunc
	metricsHandlerFunc          http.HandlerFunc
	pingHandlerFunc             http.HandlerFunc
	pprofHandlerFunc            http.HandlerFunc
	statusHandlerFunc           http.HandlerFunc
	notFoundHandlerFunc         http.HandlerFunc
	methodNotAllowedHandlerFunc http.HandlerFunc
	panicHandlerFunc            http.HandlerFunc
	redactFn                    RedactFn
	middleware                  []MiddlewareFn
	disableDefaultRouteLogger   map[DefaultRoute]bool
	disableRouteLogger          bool
	logger                      *slog.Logger
	shutdownWaitGroup           *sync.WaitGroup
	shutdownSignalChan          chan struct{}
	httpresp                    *httputil.HTTPResp
}

// defaultConfig returns the default configuration for the HTTP server.
func defaultConfig() *config {
	logger := slog.Default()

	cfg := &config{
		router:                    httprouter.New(),
		serverAddr:                ":8017",
		traceIDHeaderName:         traceid.DefaultHeader,
		serverReadHeaderTimeout:   1 * time.Minute,
		serverReadTimeout:         1 * time.Minute,
		serverWriteTimeout:        1 * time.Minute,
		shutdownTimeout:           30 * time.Second,
		defaultEnabledRoutes:      nil,
		redactFn:                  redact.HTTPData,
		middleware:                []MiddlewareFn{},
		disableDefaultRouteLogger: make(map[DefaultRoute]bool, len(allDefaultRoutes())),
		logger:                    logger,
		shutdownWaitGroup:         &sync.WaitGroup{},
		shutdownSignalChan:        make(chan struct{}),
		httpresp:                  httputil.NewHTTPResp(logger),
	}

	cfg.pprofHandlerFunc = profiling.PProfHandler
	cfg.indexHandlerFunc = cfg.defaultIndexHandler
	cfg.ipHandlerFunc = cfg.defaultIPHandler(GetPublicIPDefaultFunc())
	cfg.metricsHandlerFunc = cfg.notImplementedHandler()
	cfg.pingHandlerFunc = cfg.defaultPingHandler()
	cfg.statusHandlerFunc = cfg.defaultStatusHandler()
	cfg.notFoundHandlerFunc = cfg.defaultNotFoundHandlerFunc()
	cfg.methodNotAllowedHandlerFunc = cfg.defaultMethodNotAllowedHandlerFunc()
	cfg.panicHandlerFunc = cfg.defaultPanicHandlerFunc()

	return cfg
}

// isIndexRouteEnabled checks if the index route is enabled in the configuration.
func (c *config) isIndexRouteEnabled() bool {
	return slices.Contains(c.defaultEnabledRoutes, IndexRoute)
}

// validateAddr checks if a http server bind address is valid.
func validateAddr(addr string) error {
	addrErr := fmt.Errorf("invalid http server address: %s", addr)

	if !strings.Contains(addr, ":") {
		return addrErr
	}

	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return addrErr
	}

	port := parts[1]
	if port == "" {
		return addrErr
	}

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return addrErr
	}

	if portInt < 1 || portInt > math.MaxUint16 {
		return addrErr
	}

	return nil
}

// commonMiddleware returns the common middleware for all routes.
func (c *config) commonMiddleware(noRouteLogger bool, rTimeout time.Duration) []MiddlewareFn {
	middleware := []MiddlewareFn{}

	if !c.disableRouteLogger && !noRouteLogger {
		middleware = append(middleware, LoggerMiddlewareFn)
	}

	timeout := c.requestTimeout
	if rTimeout > 0 {
		timeout = rTimeout
	}

	if timeout > 0 {
		timeoutMiddlewareFn := func(_ MiddlewareArgs, next http.Handler) http.Handler {
			return http.TimeoutHandler(next, timeout, timeoutMessage)
		}

		middleware = append(middleware, timeoutMiddlewareFn)
	}

	return append(middleware, c.middleware...)
}

// setRouter sets the router's default handlers if they are not already set.
func (c *config) setRouter(_ context.Context) {
	l := c.logger
	middleware := c.commonMiddleware(false, 0)

	if c.router.NotFound == nil {
		c.router.NotFound = ApplyMiddleware(
			MiddlewareArgs{
				Path:              "404",
				Description:       http.StatusText(http.StatusNotFound),
				TraceIDHeaderName: c.traceIDHeaderName,
				RedactFunc:        c.redactFn,
				Logger:            l,
			},
			c.notFoundHandlerFunc,
			middleware...,
		)
	}

	if c.router.MethodNotAllowed == nil {
		c.router.MethodNotAllowed = ApplyMiddleware(
			MiddlewareArgs{
				Path:              "405",
				Description:       http.StatusText(http.StatusMethodNotAllowed),
				TraceIDHeaderName: c.traceIDHeaderName,
				RedactFunc:        c.redactFn,
				Logger:            l,
			},
			c.methodNotAllowedHandlerFunc,
			middleware...,
		)
	}

	if c.router.PanicHandler == nil {
		c.router.PanicHandler = func(w http.ResponseWriter, r *http.Request, p any) {
			c.logger.With(
				slog.Any("error", p),
				slog.String("stacktrace", string(debug.Stack())),
			).Error("panic")
			ApplyMiddleware(
				MiddlewareArgs{
					Path:              "500",
					Description:       http.StatusText(http.StatusInternalServerError),
					TraceIDHeaderName: c.traceIDHeaderName,
					RedactFunc:        c.redactFn,
					Logger:            l,
				},
				c.panicHandlerFunc,
				middleware...,
			).ServeHTTP(w, r)
		}
	}
}

// defaultIndexHandler returns the default index handler.
func (c *config) defaultIndexHandler(routes []Route) http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			data := &Index{Routes: routes}
			c.httpresp.SendJSON(r.Context(), w, http.StatusOK, data)
		},
	)
}

// defaultIPHandler returns the default /ip handler.
func (c *config) defaultIPHandler(fn GetPublicIPFunc) http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			status := http.StatusOK

			ip, err := fn(r.Context())
			if err != nil {
				status = http.StatusFailedDependency
			}

			c.httpresp.SendText(r.Context(), w, status, ip)
		},
	)
}

// defaultPingHandler returns the default /ping handler.
func (c *config) defaultPingHandler() http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			c.httpresp.SendStatus(r.Context(), w, http.StatusOK)
		},
	)
}

// defaultStatusHandler returns the default /status handler.
func (c *config) defaultStatusHandler() http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			c.httpresp.SendStatus(r.Context(), w, http.StatusOK)
		},
	)
}

// notImplementedHandler returns a 501 Not Implemented response.
func (c *config) notImplementedHandler() http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			c.httpresp.SendStatus(r.Context(), w, http.StatusNotImplemented)
		},
	)
}

// defaultNotFoundHandlerFunc returns the default 404 Not Found handler function.
func (c *config) defaultNotFoundHandlerFunc() http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			c.httpresp.SendStatus(r.Context(), w, http.StatusNotFound)
		},
	)
}

// defaultMethodNotAllowedHandlerFunc returns the default 405 Method Not Allowed handler function.
func (c *config) defaultMethodNotAllowedHandlerFunc() http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			c.httpresp.SendStatus(r.Context(), w, http.StatusMethodNotAllowed)
		},
	)
}

// defaultPanicHandlerFunc returns the default panic handler function.
func (c *config) defaultPanicHandlerFunc() http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			c.httpresp.SendStatus(r.Context(), w, http.StatusInternalServerError)
		},
	)
}
