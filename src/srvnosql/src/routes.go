package main

import "github.com/julienschmidt/httprouter"

// Route contains the HTTP route description
type Route struct {
	Method      string            `json:"method"`      // HTTP method
	Path        string            `json:"path"`        // URL path
	Handle      httprouter.Handle `json:"-"`           // Handler function
	Description string            `json:"description"` // Description
}

// Routes is a list of HTTP routes
type Routes []Route

// HTTP routes
var routes = Routes{
	Route{
		"GET",
		"/status",
		statusHandler,
		"Check this service status.",
	},
	Route{
		"GET",
		"/mongodbstatus",
		mongodbStatusHandler,
		"Check MongoDB status.",
	},
	Route{
		"GET",
		"/elasticsearchstatus",
		elasticsearchStatusHandler,
		"Check ElasticSearch status.",
	},
}
