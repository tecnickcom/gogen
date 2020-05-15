package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func initProgramTest() error {
	cfgParams, err := getConfigParams()
	if err != nil {
		return err
	}
	appParams = &cfgParams
	err = checkParams(appParams)
	if err != nil {
		return err
	}
	return nil
}

func TestIndexHandler(t *testing.T) {
	err := initProgramTest()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	status := http.StatusOK
	rw := httptest.NewRecorder()
	hr := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8017/", nil)
	indexHandler(rw, hr)
	if rw.Code != status {
		t.Errorf("Expected %d, got %d", status, rw.Code)
	}
}

func TestPingHandler(t *testing.T) {
	err := initProgramTest()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	status := http.StatusOK
	rw := httptest.NewRecorder()
	hr := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8017/ping", nil)
	pingHandler(rw, hr)
	if rw.Code != status {
		t.Errorf("Expected %d, got %d", status, rw.Code)
	}
}

func TestStatusHandler(t *testing.T) {
	err := initProgramTest()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	status := http.StatusServiceUnavailable
	rw := httptest.NewRecorder()
	hr := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8017/status", nil)
	statusHandler(rw, hr)
	if rw.Code != status {
		t.Errorf("Expected %d, got %d", status, rw.Code)
	}
}

func TestMetricsHandler(t *testing.T) {
	err := initProgramTest()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	status := http.StatusOK
	rw := httptest.NewRecorder()
	hr := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8017/metrics", nil)
	metricsHandler(rw, hr)
	if rw.Code != status {
		t.Errorf("Expected %d, got %d", status, rw.Code)
	}
}

func TestPprofHandler(t *testing.T) {
	err := initProgramTest()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	status := http.StatusOK
	rw := httptest.NewRecorder()
	hr := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8017/pprof", nil)
	pprofHandler(rw, hr)
	if rw.Code != status {
		t.Errorf("Expected %d, got %d", status, rw.Code)
	}
}

func TestIsProxyAlive(t *testing.T) {
	err := initProgramTest()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, "OK")
	}))
	defer ts.Close()
	appParams.proxyURL, err = url.Parse(ts.URL)
	if err != nil {
		t.Errorf("An error was not expected while parsing URL: %v", err)
	}
	err = isProxyAlive()
	if err != nil {
		t.Errorf("An error was not expected: %v", err)
	}
}
