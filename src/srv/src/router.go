package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

// stopServerChan is the channel used to stop the server
var stopServerChan chan os.Signal

// closeServer quietly closes the server
func closeServer(s *http.Server) {
	_ = s.Close()
}

// startServer starts the HTTP server
func startServer(address string, tlsdata *TLSData) (err error) {
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
	stats.Increment("http.server.start")

	// initialize the stopping channel
	stopServerChan = make(chan os.Signal)
	defer close(stopServerChan)

	// subscribe to SIGINT signals
	signal.Notify(stopServerChan, os.Interrupt)

	server := &http.Server{
		TLSConfig:    tlsdata.tlsConfig,
		Addr:         address,
		Handler:      router,
		ErrorLog:     stdLogger,
		ReadTimeout:  time.Minute,
		WriteTimeout: time.Minute,
	}
	defer closeServer(server)

	go func() {
		// wait for SIGINT
		if sig := <-stopServerChan; sig == nil {
			return
		}

		log.WithFields(log.Fields{
			"address": address,
		}).Info("shutting down server")

		// shut down gracefully, but wait no longer than specified timeout before halting
		ctx, cancel := context.WithTimeout(context.Background(), ServerShutdownTimeout*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	if tlsdata.Enabled {
		err = server.ListenAndServeTLS("", "")
	} else {
		err = server.ListenAndServe()
	}

	if err.Error() == "http: server closed" {
		log.WithFields(log.Fields{
			"address": address,
		}).Info("server stopped")
		stats.Increment("http.server.stop")
		return nil
	}

	return fmt.Errorf("server has stopped: %v", err)
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

// sendResponse sends the HTTP response in JSON format
func sendResponse(hw http.ResponseWriter, hr *http.Request, ps httprouter.Params, code int, data interface{}) {

	nowTime := time.Now().UTC()

	response := Response{
		Program:   ProgramName,
		Version:   ProgramVersion,
		Release:   ProgramRelease,
		URL:       appParams.serverAddress,
		DateTime:  nowTime.Format(time.RFC3339),
		Timestamp: nowTime.UnixNano(),
		Status:    getStatus(code),
		Code:      code,
		Message:   http.StatusText(code),
		Data:      data,
	}

	// log request
	if code >= 400 {
		log.WithFields(log.Fields{
			"IP":        hr.RemoteAddr,
			"UserAgent": hr.UserAgent(),
			"type":      hr.Method,
			"URI":       hr.RequestURI,
			"query":     hr.URL.Query(),
			"code":      code,
			"err":       data,
		}).Error("Request")
	} else {
		log.WithFields(log.Fields{
			"IP":        hr.RemoteAddr,
			"UserAgent": hr.UserAgent(),
			"type":      hr.Method,
			"URI":       hr.RequestURI,
			"query":     hr.URL.Query(),
			"code":      code,
		}).Info("Request")
	}

	// send JSON response
	setHeaders(hw, "application/json", code)
	err := sendJSONEncode(hw, response)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("unable to send JSON response")
	}
}
