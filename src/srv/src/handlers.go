package main

import (
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

var startTime = time.Now()

// index returns a list of available routes
func indexHandler(rw http.ResponseWriter, hr *http.Request, ps httprouter.Params) {
	stats.Increment("http.index.in")
	defer stats.Increment("http.index.out")
	log.Debug("handler: indexHandler")
	type info struct {
		Duration float64 `json:"duration"` // elapsed time since service start [seconds]
		Entries  Routes  `json:"routes"`   // available routes (http entry points)
	}
	sendResponse(rw, hr, ps, http.StatusOK, info{
		Duration: time.Since(startTime).Seconds(),
		Entries:  routes,
	})
}

// PingHandler ping back the response
func pingHandler(rw http.ResponseWriter, hr *http.Request, ps httprouter.Params) {
	stats.Increment("http.ping.in")
	defer stats.Increment("http.ping.out")
	log.Debug("handler: ping")
	sendResponse(rw, hr, ps, http.StatusOK, "OK")
}

// statusHandler returns the status of the service
func statusHandler(rw http.ResponseWriter, hr *http.Request, ps httprouter.Params) {
	stats.Increment("http.status.in")
	defer stats.Increment("http.status.out")
	log.Debug("handler: status")
	type info struct {
		Duration float64 `json:"duration"` // elapsed time since service start [seconds]
		Service  string  `json:"service"`  // error message
		Proxy    string  `json:"proxy"`    // proxy status
		Mysql    string  `json:"mysql"`    // mysql database status
		Mongo    string  `json:"mongo"`    // dmongo status
		Elastic  string  `json:"elastic"`  // elastic status
	}
	resp := &info{
		Duration: time.Since(startTime).Seconds(),
		Service:  "OK",
		Proxy:    "OK",
		Mysql:    "OK",
		Mongo:    "OK",
		Elastic:  "OK",
	}
	status := http.StatusOK
	err := isProxyAlive()
	if err != nil {
		resp.Proxy = err.Error()
		status = http.StatusServiceUnavailable
	}
	err = isMysqlAlive()
	if err != nil {
		resp.Mysql = err.Error()
		status = http.StatusServiceUnavailable
	}
	err = isMongodbAlive()
	if err != nil {
		resp.Mongo = err.Error()
		status = http.StatusServiceUnavailable
	}
	err = isElasticsearchAlive()
	if err != nil {
		resp.Elastic = err.Error()
		status = http.StatusServiceUnavailable
	}
	sendResponse(rw, hr, ps, status, resp)
}

// proxyHandler forward the request to the proxy provisioning API (reverse proxy)
func proxyHandler(rw http.ResponseWriter, hr *http.Request, ps httprouter.Params) {
	stats.Increment("http.proxy.in")
	defer stats.Increment("http.proxy.out")
	log.Debug("Handler: proxyHandler")

	proxy := httputil.NewSingleHostReverseProxy(appParams.proxyURL)
	hr.URL.Host = appParams.proxyURL.Host
	hr.URL.Scheme = appParams.proxyURL.Scheme
	hr.URL.Path = ps.ByName("path")
	hr.Header.Set("X-Forwarded-Host", hr.Header.Get("Host"))
	hr.Host = appParams.proxyURL.Host

	log.WithFields(log.Fields{
		"IP":        hr.RemoteAddr,
		"UserAgent": hr.UserAgent(),
		"type":      hr.Method,
		"URI":       hr.RequestURI,
		"query":     hr.URL.Query(),
	}).Info("Request proxy")

	proxy.ServeHTTP(rw, hr)
}

// isProxyAlive check if the proxy URL is reacheable
func isProxyAlive() error {
	_, err := http.Get(appParams.proxyURL.String())
	return err
}
