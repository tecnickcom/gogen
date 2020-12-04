package httpserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/nexmoinc/gosrvlib/pkg/httpserver/route"
	"github.com/nexmoinc/gosrvlib/pkg/ipify"
	"github.com/nexmoinc/gosrvlib/pkg/profiling"
	"github.com/nexmoinc/gosrvlib/pkg/traceid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	defaultMetricsHandler = promhttp.Handler().ServeHTTP
	defaultPprofHandler   = profiling.PProfHandler
)

// IndexHandlerFunc is a type alias for the route index function
type IndexHandlerFunc func(routes []route.Route) http.HandlerFunc

// GetPublicIPFunc is a type alias for function to get public IP of the service
type GetPublicIPFunc func(ctx context.Context) (string, error)

// GetPublicIPDefaultFunc returns the GetPublicIP function for a default ipify client
func GetPublicIPDefaultFunc() GetPublicIPFunc {
	c, _ := ipify.NewClient() // no errors are returned with default values
	return c.GetPublicIP
}

func defaultConfig() *config {
	return &config{
		router:               defaultRouter(),
		serverAddr:           ":8017",
		serverReadTimeout:    1 * time.Minute,
		serverWriteTimeout:   1 * time.Minute,
		shutdownTimeout:      30 * time.Second,
		defaultEnabledRoutes: nil,
		indexHandlerFunc:     defaultIndexHandler,
		ipHandlerFunc:        defaultIPHandler(GetPublicIPDefaultFunc()),
		metricsHandlerFunc:   defaultMetricsHandler,
		pingHandlerFunc:      defaultPingHandler,
		pprofHandlerFunc:     defaultPprofHandler,
		statusHandlerFunc:    defaultStatusHandler,
		traceIDHeaderName:    traceid.DefaultHeader,
	}
}

type config struct {
	router               Router
	serverAddr           string
	serverReadTimeout    time.Duration
	serverWriteTimeout   time.Duration
	shutdownTimeout      time.Duration
	tlsConfig            *tls.Config
	defaultEnabledRoutes []defaultRoute
	indexHandlerFunc     IndexHandlerFunc
	ipHandlerFunc        http.HandlerFunc
	metricsHandlerFunc   http.HandlerFunc
	pingHandlerFunc      http.HandlerFunc
	pprofHandlerFunc     http.HandlerFunc
	statusHandlerFunc    http.HandlerFunc
	traceIDHeaderName    string
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

	if c.ipHandlerFunc == nil {
		return fmt.Errorf("ip handler is required")
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
