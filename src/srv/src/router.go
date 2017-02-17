package main

import (
	"fmt"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
)

// server is the HTTP server
var server *http.Server

// start the HTTP server
func startServer(address string) (err error) {
	log.Info("setting http router")
	router := httprouter.New()

	// set error handlers
	router.NotFound = http.HandlerFunc(func(rw http.ResponseWriter, hr *http.Request) { // 404
		sendResponse(rw, hr, nil, http.StatusNotFound, "invalid end point")
	})
	router.MethodNotAllowed = http.HandlerFunc(func(rw http.ResponseWriter, hr *http.Request) { // 405
		sendResponse(rw, hr, nil, http.StatusMethodNotAllowed, "the request cannot be routed")
	})
	router.PanicHandler = func(rw http.ResponseWriter, hr *http.Request, p interface{}) { // 500
		sendResponse(rw, hr, nil, http.StatusInternalServerError, "internal error")
	}

	// index handler
	router.GET("/", indexHandler)

	// set end points and handlers
	for _, route := range routes {
		router.Handle(route.Method, route.Path, route.Handle)
	}

	log.WithFields(log.Fields{
		"address": address,
	}).Info("starting http server")

	server = &http.Server{
		Addr:     address,
		Handler:  router,
		ErrorLog: stdLogger,
	}
	defer server.Close()

	return fmt.Errorf("unable to start the HTTP server: %v", server.ListenAndServe())
}

// setHeaders set the default headers
func setHeaders(hw http.ResponseWriter, contentType string, code int) {
	hw.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	hw.Header().Set("Pragma", "no-cache")
	hw.Header().Set("Expires", "0")
	hw.Header().Set("Content-Type", contentType)
	hw.WriteHeader(code)

	// count HTTP status
	stats.Increment(fmt.Sprintf("httpstatus.%d", code))
}

// send the HTTP response in JSON format
func sendResponse(hw http.ResponseWriter, hr *http.Request, ps httprouter.Params, code int, data interface{}) {

	setHeaders(hw, "application/json", code)

	nowTime := time.Now().UTC()

	response := Response{
		Program:   ProgramName,
		Version:   ProgramVersion,
		Release:   ProgramRelease,
		DateTime:  nowTime.Format(time.RFC3339),
		Timestamp: nowTime.UnixNano(),
		Status:    getStatus(code),
		Code:      code,
		Message:   http.StatusText(code),
		Data:      data,
	}

	// log request
	if code == 500 {
		log.WithFields(log.Fields{
			"type":  hr.Method,
			"URI":   hr.RequestURI,
			"query": hr.URL.Query(),
			"code":  code,
			"err":   data.(string),
		}).Error("request")
	} else {
		log.WithFields(log.Fields{
			"type":  hr.Method,
			"URI":   hr.RequestURI,
			"query": hr.URL.Query(),
			"code":  code,
		}).Info("request")
	}

	// send JSON response
	err := sendJSONEncode(hw, response)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("unable to send JSON response")
	}
}
