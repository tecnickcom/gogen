package main

import (
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

var startTime = time.Now()

// return a list of available routes
func indexHandler(rw http.ResponseWriter, hr *http.Request, ps httprouter.Params) {
	stats.Increment("http.index.in")
	type info struct {
		Duration float64 `json:"duration"` // elapsed time since last passwor drequest or service start
		Entries  Routes  `json:"routes"`   // available routes (http entry points)
	}
	sendResponse(rw, hr, ps, http.StatusOK, info{
		Duration: time.Since(startTime).Seconds(),
		Entries:  routes,
	})
	stats.Increment("http.index.out")
}

// returns the status of the service
func statusHandler(rw http.ResponseWriter, hr *http.Request, ps httprouter.Params) {
	stats.Increment("http.status.in")
	type info struct {
		Duration float64 `json:"duration"` // elapsed time since last request in seconds
		Message  string  `json:"message"`  // error message
	}
	status := http.StatusOK
	message := "The service is healthy"
	sendResponse(rw, hr, ps, status, info{
		Duration: time.Since(startTime).Seconds(),
		Message:  message,
	})
	stats.Increment("http.status.out")
}
