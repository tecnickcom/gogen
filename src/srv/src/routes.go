package main

import "net/http"

// Route contains the HTTP route description
type Route struct {
	Method      string           `json:"method"`      // HTTP method
	Path        string           `json:"path"`        // URL path
	Handler     http.HandlerFunc `json:"-"`           // Handler function
	Description string           `json:"description"` // Description
}

// Routes is a list of HTTP routes
type Routes []Route

// HTTP routes
var routes = Routes{
	Route{
		"GET",
		"/ping",
		pingHandler,
		"Ping this service.",
	},
	Route{
		"GET",
		"/status",
		statusHandler,
		"Check this service health status.",
	},
	Route{
		"POST",
		"/auth/login",
		loginHandler,
		"Login to get a JWT token. The post body must be a JSON: '{\"username\":\"YOUR_USERNAME\",\"password\":\"YOUR_PASSWORD\"}'",
	},
	Route{
		"GET",
		"/auth/refresh",
		renewJwtHandler,
		"Renew the JWT token before expiration.",
	},
	Route{
		"GET",
		"/proxy/*path",
		proxyHandler,
		"Handle requests using the proxy provisioning API.",
	},
}
