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
	type info struct {
		Duration float64 `json:"duration"` // elapsed time since service start [seconds]
		Message  string  `json:"message"`  // error message
	}
	status := http.StatusOK
	message := "The service is healthy"
	sendResponse(rw, hr, ps, status, info{
		Duration: time.Since(startTime).Seconds(),
		Message:  message,
	})
}

// mongodbStatusHandler returns the status of MongoDB
func mongodbStatusHandler(rw http.ResponseWriter, hr *http.Request, ps httprouter.Params) {
	stats.Increment("http.mongodb.status.in")
	defer stats.Increment("http.mongodb.status.out")
	log.Debug("Handler: mongodbstatus")
	type info struct {
		Duration float64 `json:"duration"` // elapsed time since service start [seconds]
		Message  string  `json:"message"`  // error message
	}
	status := http.StatusOK
	message := "MongoDB is alive"

	session := appParams.mongodb.session.Copy()
	defer session.Close()

	dbs, err := session.DatabaseNames()
	if err != nil && len(dbs) <= 0 {
		status = http.StatusServiceUnavailable
		message = "MongoDB is not working"
		log.WithFields(log.Fields{
			"error":           err,
			"mongodbAddress":  appParams.mongodb.Address,
			"mongodbDatabase": appParams.mongodb.Database,
		}).Error("MongoDB")
	}

	sendResponse(rw, hr, ps, status, info{
		Duration: time.Since(startTime).Seconds(),
		Message:  message,
	})
}

// elasticsearchStatusHandler returns the status of ElasticSearch
func elasticsearchStatusHandler(rw http.ResponseWriter, hr *http.Request, ps httprouter.Params) {
	stats.Increment("http.elasticsearch.status.in")
	defer stats.Increment("http.elasticsearch.status.out")
	log.Debug("Handler: elasticsearchstatus")
	type info struct {
		Duration float64 `json:"duration"` // elapsed time since service start [seconds]
		Message  string  `json:"message"`  // error message
	}
	status := http.StatusOK
	message := "ElasticSearch is alive - VERSION: "

	log.WithFields(log.Fields{
		"elasticsearchURL":   appParams.elasticsearch.URL,
		"elasticsearchIndex": appParams.elasticsearch.Index,
		"ctx":                appParams.elasticsearch.ctx,
	}).Info("DEBUG")

	esinfo, code, err := appParams.elasticsearch.client.Ping(appParams.elasticsearch.URL).Do(appParams.elasticsearch.ctx)
	if err != nil || code != 200 {
		status = http.StatusServiceUnavailable
		message = "ElasticSearch is not working"
		log.WithFields(log.Fields{
			"error":              err,
			"elasticsearchURL":   appParams.elasticsearch.URL,
			"elasticsearchIndex": appParams.elasticsearch.Index,
		}).Error("ElasticSearch")
	} else {
		message += esinfo.Version.Number
	}

	sendResponse(rw, hr, ps, status, info{
		Duration: time.Since(startTime).Seconds(),
		Message:  message,
	})
}
