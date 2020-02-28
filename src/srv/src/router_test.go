package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetHeaders(t *testing.T) {
	rw := httptest.NewRecorder()
	setHeaders(rw, "application/test", http.StatusOK)
	if rw.Code != http.StatusOK {
		t.Error(fmt.Errorf("Expected %d, got %d", http.StatusOK, rw.Code))
	}
	hdr := rw.Header().Get("Content-Type")
	if hdr != "application/test" {
		t.Error(fmt.Errorf("Expected 'application/test', got %s", hdr))
	}
}

func TestSendResponseOK(t *testing.T) {
	rw := httptest.NewRecorder()
	hr := httptest.NewRequest("GET", "http://example.com", nil)
	code := http.StatusOK
	data := "TEST STRING"
	sendResponse(rw, hr, code, data)
	if rw.Code != code {
		t.Error(fmt.Errorf("Expected %d, got %d", code, rw.Code))
	}
	hdr := rw.Header().Get("Content-Type")
	if hdr != "application/json" {
		t.Error(fmt.Errorf("Expected 'application/json', got %s", hdr))
	}
	if !strings.Contains(rw.Body.String(), `"data":"TEST STRING"`) {
		t.Error(fmt.Errorf("The resulting body is not correct: %v", rw.Body.String()))
	}
}

func TestSendResponseInternalServerError(t *testing.T) {
	rw := httptest.NewRecorder()
	hr := httptest.NewRequest("GET", "http://example.com", nil)
	code := http.StatusInternalServerError
	data := "TEST STRING"
	sendResponse(rw, hr, code, data)
	if rw.Code != code {
		t.Error(fmt.Errorf("Expected %d, got %d", code, rw.Code))
	}
	hdr := rw.Header().Get("Content-Type")
	if hdr != "application/json" {
		t.Error(fmt.Errorf("Expected 'application/json', got %s", hdr))
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
	code := http.StatusOK
	data := "TEST STRING"
	sendResponse(rw, hr, code, data)
	if rw.Code != code {
		t.Error(fmt.Errorf("Expected %d, got %d", code, rw.Code))
	}
}

func TestStartServerError(t *testing.T) {
	err := startServer("-1", &TLSData{})
	if err == nil {
		t.Error(fmt.Errorf("An error was expected"))
	}
}
