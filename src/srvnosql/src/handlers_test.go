package main

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
)

func TestIndexHandler(t *testing.T) {

	err := initProgramTest()
	if err != nil {
		t.Error(fmt.Errorf("Unexpected error: %v", err))
	}
	defer stats.Close()

	status := 200
	rw := httptest.NewRecorder()
	hr := httptest.NewRequest("GET", "http://127.0.0.1:8017/", nil)
	ps := httprouter.Params{}

	indexHandler(rw, hr, ps)

	if rw.Code != status {
		t.Error(fmt.Errorf("Expected %d, got %d", status, rw.Code))
	}
}

func TestStatusHandler(t *testing.T) {

	err := initProgramTest()
	if err != nil {
		t.Error(fmt.Errorf("Unexpected error: %v", err))
	}
	defer stats.Close()

	status := 200
	rw := httptest.NewRecorder()
	hr := httptest.NewRequest("GET", "http://127.0.0.1:8017/status", nil)
	ps := httprouter.Params{}

	statusHandler(rw, hr, ps)

	if rw.Code != status {
		t.Error(fmt.Errorf("Expected %d, got %d", status, rw.Code))
	}
}

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

	// initialize StatsD client
	err = initStats(appParams.stats)
	if err != nil {
		return err
	}

	// initialize MongoDB Session
	appParams.mongodb, err = initMongodbSession(appParams.mongodb)
	if err != nil {
		return err
	}

	// initialize ElasticSearch Session
	appParams.elasticsearch, err = initElasticsearchSession(appParams.elasticsearch)
	if err != nil {
		return err
	}

	return err
}
