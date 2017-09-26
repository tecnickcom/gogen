package main

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetHeaders(t *testing.T) {
	rw := httptest.NewRecorder()
	setHeaders(rw, "application/test", 200)
	if rw.Code != 200 {
		t.Error(fmt.Errorf("Expected 200, got %d", rw.Code))
	}
	if rw.HeaderMap["Content-Type"][0] != "application/test" {
		t.Error(fmt.Errorf("Expected 'application/test', got %s", rw.HeaderMap["Content-Type"][0]))
	}
}

func TestSendResponse200(t *testing.T) {
	rw := httptest.NewRecorder()
	hr := httptest.NewRequest("GET", "http://example.com", nil)
	code := 200
	data := "TEST STRING"

	sendResponse(rw, hr, nil, code, data)

	if rw.Code != 200 {
		t.Error(fmt.Errorf("Expected 200, got %d", rw.Code))
	}
	if rw.HeaderMap["Content-Type"][0] != "application/json" {
		t.Error(fmt.Errorf("Expected 'application/json', got %s", rw.HeaderMap["Content-Type"][0]))
	}
	if !strings.Contains(rw.Body.String(), `"data":"TEST STRING"`) {
		t.Error(fmt.Errorf("The resulting body is not correct: %v", rw.Body.String()))
	}
}

func TestSendResponse500(t *testing.T) {
	rw := httptest.NewRecorder()
	hr := httptest.NewRequest("GET", "http://example.com", nil)
	code := 500
	data := "TEST STRING"

	sendResponse(rw, hr, nil, code, data)

	if rw.Code != 500 {
		t.Error(fmt.Errorf("Expected 500, got %d", rw.Code))
	}
	if rw.HeaderMap["Content-Type"][0] != "application/json" {
		t.Error(fmt.Errorf("Expected 'application/json', got %s", rw.HeaderMap["Content-Type"][0]))
	}
	if !strings.Contains(rw.Body.String(), `"data":"TEST STRING"`) {
		t.Error(fmt.Errorf("The resulting body is not correct: %v", rw.Body.String()))
	}
}

func TestSendResponseError(t *testing.T) {

	oldSendJSONEncode := sendJSONEncode
	defer func() { sendJSONEncode = oldSendJSONEncode }()
	sendJSONEncode = mockSendJSONEncode

	rw := httptest.NewRecorder()
	hr := httptest.NewRequest("GET", "http://example.com", nil)
	code := 200
	data := "TEST STRING"

	sendResponse(rw, hr, nil, code, data)

	if rw.Code != 200 {
		t.Error(fmt.Errorf("Expected 200, got %d", rw.Code))
	}
}

func TestStartServerError(t *testing.T) {
	err := startServer("-1")
	if err == nil {
		t.Error(fmt.Errorf("An error was expected"))
	}
}
