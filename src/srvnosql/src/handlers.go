package main

import (
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

var startTime = time.Now()

// index returns a list of available routes
func indexHandler(rw http.ResponseWriter, hr *http.Request, ps httprouter.Params) {
	stats.Increment("http.index.in")
	defer stats.Increment("http.index.out")
	log.Debug("Handler: index")
	type info struct {
		Duration float64 `json:"duration"` // elapsed time since service start [seconds]
		Entries  Routes  `json:"routes"`   // available routes (http entry points)
	}
	sendResponse(rw, hr, ps, http.StatusOK, info{
		Duration: time.Since(startTime).Seconds(),
		Entries:  routes,
	})
}

// statusHandler returns the status of the service
func statusHandler(rw http.ResponseWriter, hr *http.Request, ps httprouter.Params) {
	stats.Increment("http.status.in")
	defer stats.Increment("http.status.out")
	log.Debug("Handler: status")
	type dberror struct {
		MongoDB       error `json:"mongodb"`       // MongoDB status
		Elasticsearch error `json:"elasticsearch"` // Elasticsearch status
	}
	type info struct {
		Duration float64 `json:"duration"` // elapsed time since service start [seconds]
		Message  string  `json:"message"`  // error message
		DBError  dberror `json:"dberror"`  // database error message
	}
	status := http.StatusOK
	message := "The HTTP service is healthy."

	mongodbErr := isMongodbAlive()
	elasticsearchErr := isElasticsearchAlive()

	if mongodbErr != nil || elasticsearchErr != nil {
		status = http.StatusServiceUnavailable
	}

	sendResponse(rw, hr, ps, status, info{
		Duration: time.Since(startTime).Seconds(),
		Message:  message,
		DBError: dberror{
			MongoDB:       mongodbErr,
			Elasticsearch: elasticsearchErr,
		},
	})
}
