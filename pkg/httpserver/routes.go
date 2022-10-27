package httpserver

import (
	"net/http"
)

// DefaultRoute is the type for the default route names.
type DefaultRoute string

const (
	// IndexRoute is the identifier to enable the index handler.
	IndexRoute DefaultRoute = "index"
	indexPath  string       = "/"

	// IPRoute is the identifier to enable the ip handler.
	IPRoute       DefaultRoute = "ip"
	ipHandlerPath string       = "/ip"

	// MetricsRoute is the identifier to enable the metrics handler.
	MetricsRoute       DefaultRoute = "metrics"
	metricsHandlerPath string       = "/metrics"

	// PingRoute is the identifier to enable the ping handler.
	PingRoute       DefaultRoute = "ping"
	pingHandlerPath string       = "/ping"

	// PprofRoute is the identifier to enable the pprof handler.
	PprofRoute       DefaultRoute = "pprof"
	pprofHandlerPath string       = "/pprof/*option"

	// StatusRoute is the identifier to enable the status handler.
	StatusRoute       DefaultRoute = "status"
	statusHandlerPath string       = "/status"
)

func allDefaultRoutes() []DefaultRoute {
	return []DefaultRoute{
		IndexRoute,
		IPRoute,
		MetricsRoute,
		PingRoute,
		PprofRoute,
		StatusRoute,
	}
}

func newDefaultRoutes(cfg *config) []Route {
	routes := make([]Route, 0, len(cfg.defaultEnabledRoutes)+1)

	for _, id := range cfg.defaultEnabledRoutes {
		_, disableLogger := cfg.disableDefaultRouteLogger[id]

		switch id {
		case IndexRoute:
			// The index route needs to access all the routes bound to the handler.
		case IPRoute:
			routes = append(routes, Route{
				Method:        http.MethodGet,
				Path:          ipHandlerPath,
				Handler:       cfg.ipHandlerFunc,
				DisableLogger: disableLogger,
				Description:   "Returns the public IP address of this service instance.",
			})
		case MetricsRoute:
			routes = append(routes, Route{
				Method:        http.MethodGet,
				Path:          metricsHandlerPath,
				Handler:       cfg.metricsHandlerFunc,
				DisableLogger: disableLogger,
				Description:   "Returns Prometheus metrics.",
			})
		case PingRoute:
			routes = append(routes, Route{
				Method:        http.MethodGet,
				Path:          pingHandlerPath,
				Handler:       cfg.pingHandlerFunc,
				DisableLogger: disableLogger,
				Description:   "Ping this service.",
			})
		case PprofRoute:
			routes = append(routes, Route{
				Method:        http.MethodGet,
				Path:          pprofHandlerPath,
				Handler:       cfg.pprofHandlerFunc,
				DisableLogger: disableLogger,
				Description:   "Returns pprof data for the selected profile.",
			})
		case StatusRoute:
			routes = append(routes, Route{
				Method:        http.MethodGet,
				Path:          statusHandlerPath,
				Handler:       cfg.statusHandlerFunc,
				DisableLogger: disableLogger,
				Description:   "Check this service health status.",
			})
		}
	}

	return routes
}
