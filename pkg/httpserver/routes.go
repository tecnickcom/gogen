package httpserver

import (
	"net/http"

	"github.com/nexmoinc/gosrvlib/pkg/httpserver/route"
)

type defaultRoute string

const (
	// IndexRoute is the identifier to enable the index handler
	IndexRoute defaultRoute = "index"
	indexPath  string       = "/"

	// IPRoute is the identifier to enable the ip handler
	IPRoute       defaultRoute = "ip"
	ipHandlerPath string       = "/ip"

	// MetricsRoute is the identifier to enable the metrics handler
	MetricsRoute       defaultRoute = "metrics"
	metricsHandlerPath string       = "/metrics"

	// PingRoute is the identifier to enable the ping handler
	PingRoute       defaultRoute = "ping"
	pingHandlerPath string       = "/ping"

	// PprofRoute is the identifier to enable the pprof handler
	PprofRoute       defaultRoute = "pprof"
	pprofHandlerPath string       = "/pprof/*option"

	// StatusRoute is the identifier to enable the status handler
	StatusRoute       defaultRoute = "status"
	statusHandlerPath string       = "/status"
)

var allDefaultRoutes = []defaultRoute{IndexRoute, IPRoute, MetricsRoute, PingRoute, PprofRoute, StatusRoute}

func newDefaultRoutes(cfg *config) []route.Route {
	routes := make([]route.Route, 0)

	// The index route is not included here because of the need of accessing all the routes bound to the handler
	for _, id := range cfg.defaultEnabledRoutes {
		switch id {
		case IPRoute:
			routes = append(routes, route.Route{
				Method:      http.MethodGet,
				Path:        ipHandlerPath,
				Handler:     cfg.ipHandlerFunc,
				Description: "Returns the public IP address of this service instance.",
			})
		case MetricsRoute:
			routes = append(routes, route.Route{
				Method:      http.MethodGet,
				Path:        metricsHandlerPath,
				Handler:     cfg.metricsHandlerFunc,
				Description: "Returns Prometheus metrics.",
			})
		case PingRoute:
			routes = append(routes, route.Route{
				Method:      http.MethodGet,
				Path:        pingHandlerPath,
				Handler:     cfg.pingHandlerFunc,
				Description: "Ping this service.",
			})
		case PprofRoute:
			routes = append(routes, route.Route{
				Method:      http.MethodGet,
				Path:        pprofHandlerPath,
				Handler:     cfg.pprofHandlerFunc,
				Description: "Returns pprof data for the selected profile.",
			})
		case StatusRoute:
			routes = append(routes, route.Route{
				Method:      http.MethodGet,
				Path:        statusHandlerPath,
				Handler:     cfg.statusHandlerFunc,
				Description: "Check this service health status.",
			})
		}
	}

	return routes
}
