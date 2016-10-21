package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
)

// start the HTTP server
func startServer(address string) error {
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
	router.GET("/", index)

	// set end points and handlers
	for _, route := range routes {
		router.Handle(route.Method, route.Path, route.Handle)
	}

	log.WithFields(log.Fields{
		"address": address,
	}).Info("starting http server")
	return fmt.Errorf("unable to start the HTTP server: %v", http.ListenAndServe(address, router))
}

// send the HTTP response in JSON format
func sendResponse(hw http.ResponseWriter, hr *http.Request, ps httprouter.Params, code int, data interface{}) {
	hw.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	hw.Header().Set("Pragma", "no-cache")
	hw.Header().Set("Expires", "0")
	hw.Header().Set("Content-Type", "application/json")
	hw.WriteHeader(code)

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

	// count HTTP statuses
	stats.Increment(fmt.Sprintf("httpstatus.%s", response.Status))

	// log request
	log.WithFields(log.Fields{
		"type": hr.Method,
		"URI":  hr.RequestURI,
		"code": code,
	}).Info("request")

	// send response as JSON
	if err := json.NewEncoder(hw).Encode(response); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("unable to send the JSON response")
	}
}
