// Package jsendx implements a custom JSEND model to wrap HTTP responses in a JSON object with default fields.
package jsendx

import (
	"context"
	"net/http"
	"time"

	"github.com/Vonage/gosrvlib/pkg/healthcheck"
	"github.com/Vonage/gosrvlib/pkg/httpserver"
	"github.com/Vonage/gosrvlib/pkg/httputil"
)

const (
	okMessage = "OK"
)

// Response wraps data into a JSend compliant response.
type Response struct {
	// Program is the application name.
	Program string `json:"program"`

	// Version is the program semantic version (e.g. 1.2.3).
	Version string `json:"version"`

	// Release is the program build number that is appended to the version.
	Release string `json:"release"`

	// DateTime is the human-readable date and time when the response is sent.
	DateTime string `json:"datetime"`

	// Timestamp is the machine-readable UTC timestamp in nanoseconds since EPOCH.
	Timestamp int64 `json:"timestamp"`

	// Status code string (i.e.: error, fail, success).
	Status httputil.Status `json:"status"`

	// Code is the HTTP status code number.
	Code int `json:"code"`

	// Message is the error or general HTTP status message.
	Message string `json:"message"`

	// Data is the content payload.
	Data interface{} `json:"data"`
}

// AppInfo is a struct containing data to enrich the JSendX response.
type AppInfo struct {
	ProgramName    string
	ProgramVersion string
	ProgramRelease string
}

// RouterArgs extra arguments for the router.
type RouterArgs struct {
	// TraceIDHeaderName is the Trace ID header name.
	TraceIDHeaderName string

	// RedactFunc is the function used to redact HTTP request and response dumps in the logs.
	RedactFunc httpserver.RedactFn
}

// Wrap sends an Response object.
func Wrap(statusCode int, info *AppInfo, data interface{}) *Response {
	now := time.Now().UTC()

	return &Response{
		Program:   info.ProgramName,
		Version:   info.ProgramVersion,
		Release:   info.ProgramRelease,
		DateTime:  now.Format(time.RFC3339),
		Timestamp: now.UnixNano(),
		Status:    httputil.Status(statusCode),
		Code:      statusCode,
		Message:   http.StatusText(statusCode),
		Data:      data,
	}
}

// Send sends a JSON respoonse wrapped in a JSendX container.
func Send(ctx context.Context, w http.ResponseWriter, statusCode int, info *AppInfo, data interface{}) {
	httputil.SendJSON(ctx, w, statusCode, Wrap(statusCode, info, data))
}

// DefaultNotFoundHandlerFunc http handler called when no matching route is found.
func DefaultNotFoundHandlerFunc(info *AppInfo) http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			Send(r.Context(), w, http.StatusNotFound, info, "invalid endpoint")
		},
	)
}

// DefaultMethodNotAllowedHandlerFunc http handler called when a request cannot be routed.
func DefaultMethodNotAllowedHandlerFunc(info *AppInfo) http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			Send(r.Context(), w, http.StatusMethodNotAllowed, info, "the request cannot be routed")
		},
	)
}

// DefaultPanicHandlerFunc http handler to handle panics recovered from http handlers.
func DefaultPanicHandlerFunc(info *AppInfo) http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			Send(r.Context(), w, http.StatusInternalServerError, info, "internal error")
		},
	)
}

// DefaultIndexHandler returns the route index in JSendX format.
func DefaultIndexHandler(info *AppInfo) httpserver.IndexHandlerFunc {
	return func(routes []httpserver.Route) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			data := &httpserver.Index{Routes: routes}
			Send(r.Context(), w, http.StatusOK, info, data)
		}
	}
}

// DefaultIPHandler returns the route ip in JSendX format.
func DefaultIPHandler(info *AppInfo, fn httpserver.GetPublicIPFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusOK

		ip, err := fn(r.Context())
		if err != nil {
			status = http.StatusFailedDependency
		}

		Send(r.Context(), w, status, info, ip)
	}
}

// DefaultPingHandler returns a ping request in JSendX format.
func DefaultPingHandler(info *AppInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		Send(r.Context(), w, http.StatusOK, info, okMessage)
	}
}

// DefaultStatusHandler returns the server status in JSendX format.
func DefaultStatusHandler(info *AppInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		Send(r.Context(), w, http.StatusOK, info, okMessage)
	}
}

// HealthCheckResultWriter returns a new healthcheck result writer.
func HealthCheckResultWriter(info *AppInfo) healthcheck.ResultWriter {
	return func(ctx context.Context, w http.ResponseWriter, statusCode int, data interface{}) {
		Send(ctx, w, statusCode, info, data)
	}
}
