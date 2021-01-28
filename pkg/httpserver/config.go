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
)

// IndexHandlerFunc is a type alias for the route index function.
type IndexHandlerFunc func([]route.Route) http.HandlerFunc

// GetPublicIPFunc is a type alias for function to get public IP of the service.
type GetPublicIPFunc func(context.Context) (string, error)

// InstrumentHandler is a type alias for the function used to wrap an http.Handler to collect metrics.
type InstrumentHandler func(string, http.HandlerFunc) http.Handler

// GetPublicIPDefaultFunc returns the GetPublicIP function for a default ipify client.
func GetPublicIPDefaultFunc() GetPublicIPFunc {
	c, _ := ipify.NewClient() // no errors are returned with default values
	return c.GetPublicIP
}

type config struct {
	router               Router
	serverAddr           string
	serverReadTimeout    time.Duration
	serverWriteTimeout   time.Duration
	shutdownTimeout      time.Duration
	tlsConfig            *tls.Config
	instrumentHandler    InstrumentHandler
	defaultEnabledRoutes []defaultRoute
	indexHandlerFunc     IndexHandlerFunc
	ipHandlerFunc        http.HandlerFunc
	metricsHandlerFunc   http.HandlerFunc
	pingHandlerFunc      http.HandlerFunc
	pprofHandlerFunc     http.HandlerFunc
	statusHandlerFunc    http.HandlerFunc
	traceIDHeaderName    string
}

func defaultConfig() *config {
	defaultInstrumentHandler := func(path string, handler http.HandlerFunc) http.Handler { return handler }
	return &config{
		serverAddr:           ":8017",
		serverReadTimeout:    1 * time.Minute,
		serverWriteTimeout:   1 * time.Minute,
		shutdownTimeout:      30 * time.Second,
		instrumentHandler:    defaultInstrumentHandler,
		defaultEnabledRoutes: nil,
		indexHandlerFunc:     defaultIndexHandler,
		ipHandlerFunc:        defaultIPHandler(GetPublicIPDefaultFunc()),
		metricsHandlerFunc:   notImplementedHandler,
		pingHandlerFunc:      defaultPingHandler,
		pprofHandlerFunc:     profiling.PProfHandler,
		statusHandlerFunc:    defaultStatusHandler,
		traceIDHeaderName:    traceid.DefaultHeader,
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

// nolint: gocyclo
func (c *config) validate() error {
	if err := validateAddr(c.serverAddr); err != nil {
		return err
	}
	if c.shutdownTimeout == 0 {
		return fmt.Errorf("invalid shutdownTimeout")
	}
	if c.instrumentHandler == nil {
		return fmt.Errorf("instrumentHandler is required")
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
