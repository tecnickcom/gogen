package httpserver

import (
	"crypto/tls"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/nexmoinc/gosrvlib/pkg/profiling"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	metricsHandlerPath = "/metrics"
	pingHandlerPath    = "/ping"
	pprofHandlerPath   = "/pprof/*option"
	statusHandlerPath  = "/status"
)

var (
	defaultMetricsHandler = promhttp.Handler().ServeHTTP
	defaultPprofHandler   = profiling.PProfHandler
	defaultEnabledRoutes  = []defaultRoute{IndexRoute, MetricsRoute, PingRoute, PprofRoute, StatusRoute}
)

func defaultConfig() *config {
	return &config{
		defaultEnabledRoutes: defaultEnabledRoutes,
		metricsHandlerFunc:   defaultMetricsHandler,
		pingHandlerFunc:      defaultPingHandler,
		pprofHandlerFunc:     defaultPprofHandler,
		serverAddr:           ":8080",
		serverReadTimeout:    1 * time.Minute,
		serverWriteTimeout:   1 * time.Minute,
		shutdownTimeout:      30 * time.Second,
		statusHandlerFunc:    defaultStatusHandler,
		router:               defaultRouter(),
	}
}

type config struct {
	defaultEnabledRoutes []defaultRoute
	metricsHandlerFunc   http.HandlerFunc
	pingHandlerFunc      http.HandlerFunc
	pprofHandlerFunc     http.HandlerFunc
	router               Router
	serverAddr           string
	serverReadTimeout    time.Duration
	serverWriteTimeout   time.Duration
	shutdownTimeout      time.Duration
	statusHandlerFunc    http.HandlerFunc
	tlsConfig            *tls.Config
}

func (c *config) isIndexRouteEnabled() bool {
	for _, r := range c.defaultEnabledRoutes {
		if r == IndexRoute {
			return true
		}
	}
	return false
}

func (c *config) validate() error {
	if err := validateAddr(c.serverAddr); err != nil {
		return err
	}

	if c.shutdownTimeout == 0 {
		return fmt.Errorf("invalid shutdown timeout")
	}

	if c.router == nil {
		return fmt.Errorf("router is required")
	}

	if c.metricsHandlerFunc == nil {
		return fmt.Errorf("metrics handler is required")
	}

	if c.pingHandlerFunc == nil {
		return fmt.Errorf("ping handler is required")
	}

	if c.pprofHandlerFunc == nil {
		return fmt.Errorf("pprof handler is required")
	}

	if c.statusHandlerFunc == nil {
		return fmt.Errorf("status handler is required")
	}

	return nil
}

// validateAddr checks if a http server bind address is valid
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
