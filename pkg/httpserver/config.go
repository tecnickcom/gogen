package httpserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/nexmoinc/gosrvlib/pkg/ipify"
	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"github.com/nexmoinc/gosrvlib/pkg/profiling"
	"github.com/nexmoinc/gosrvlib/pkg/redact"
	"github.com/nexmoinc/gosrvlib/pkg/traceid"
	"go.uber.org/zap"
)

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

type config struct {
	router                      *httprouter.Router
	serverAddr                  string
	traceIDHeaderName           string
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
}

func defaultConfig() *config {
	return &config{
		router:                      httprouter.New(),
		serverAddr:                  ":8017",
		traceIDHeaderName:           traceid.DefaultHeader,
		serverReadHeaderTimeout:     1 * time.Minute,
		serverReadTimeout:           1 * time.Minute,
		serverWriteTimeout:          1 * time.Minute,
		shutdownTimeout:             30 * time.Second,
		defaultEnabledRoutes:        nil,
		indexHandlerFunc:            defaultIndexHandler,
		ipHandlerFunc:               defaultIPHandler(GetPublicIPDefaultFunc()),
		metricsHandlerFunc:          notImplementedHandler,
		pingHandlerFunc:             defaultPingHandler,
		pprofHandlerFunc:            profiling.PProfHandler,
		statusHandlerFunc:           defaultStatusHandler,
		notFoundHandlerFunc:         defaultNotFoundHandlerFunc,
		methodNotAllowedHandlerFunc: defaultMethodNotAllowedHandlerFunc,
		panicHandlerFunc:            defaultPanicHandlerFunc,
		redactFn:                    redact.HTTPData,
		middleware:                  []MiddlewareFn{},
		disableDefaultRouteLogger:   make(map[DefaultRoute]bool, len(allDefaultRoutes())),
	}
}

func (c *config) isIndexRouteEnabled() bool {
	for _, r := range c.defaultEnabledRoutes {
		if r == IndexRoute {
			return true
		}
	}

	return false
}

// validate the configuration.
func (c *config) validate() error {
	if err := validateAddr(c.serverAddr); err != nil {
		return err
	}

	if c.shutdownTimeout == 0 {
		return fmt.Errorf("invalid shutdownTimeout")
	}

	if c.ipHandlerFunc == nil {
		return fmt.Errorf("ipHandlerFunc is required")
	}

	if c.metricsHandlerFunc == nil {
		return fmt.Errorf("metricsHandlerFunc is required")
	}

	if c.pingHandlerFunc == nil {
		return fmt.Errorf("pingHandlerFunc is required")
	}

	if c.pprofHandlerFunc == nil {
		return fmt.Errorf("pprofHandlerFunc is required")
	}

	if c.statusHandlerFunc == nil {
		return fmt.Errorf("statusHandlerFunc is required")
	}

	if c.traceIDHeaderName == "" {
		return fmt.Errorf("traceIDHeaderName is required")
	}

	if c.router == nil {
		return fmt.Errorf("router is required")
	}

	return nil
}

// validateAddr checks if a http server bind address is valid.
func validateAddr(addr string) error {
	if !strings.Contains(addr, ":") {
		return fmt.Errorf("invalid http server address: %s", addr)
	}

	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid http server address: %s", addr)
	}

	port := parts[1]
	if port == "" {
		return fmt.Errorf("invalid http server address: %s", addr)
	}

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid http server address: %s", addr)
	}

	if portInt < 1 || portInt > math.MaxUint16 {
		return fmt.Errorf("invalid http server address: %s", addr)
	}

	return nil
}

func (c *config) commonMiddleware(noRouteLogger bool) []MiddlewareFn {
	middleware := []MiddlewareFn{}

	if !c.disableRouteLogger && !noRouteLogger {
		middleware = append(middleware, LoggerMiddlewareFn)
	}

	return append(middleware, c.middleware...)
}

func (c *config) setRouter(ctx context.Context) {
	l := logging.FromContext(ctx)
	middleware := c.commonMiddleware(false)

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
		c.router.PanicHandler = func(w http.ResponseWriter, r *http.Request, p interface{}) {
			logging.FromContext(r.Context()).Error(
				"panic",
				zap.Any("err", p),
				zap.String("stacktrace", string(debug.Stack())),
			)
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
