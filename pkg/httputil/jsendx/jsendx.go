/*
Package jsendx implements an extended JSend response model for HTTP APIs.

It solves the problem of inconsistent response envelopes by wrapping all HTTP
response payloads in a predictable JSON structure enriched with application
metadata.

This package adds fields such as program name, version, release, timestamp, and
status metadata on top of the JSend convention, making responses easier to
consume and debug in REST-style applications.

Top features:
  - consistent response envelope for success and error payloads
  - automatic enrichment with application metadata
  - reusable default handlers for not-found, method-not-allowed, panic, and index
    responses
  - easy integration with existing HTTP and healthcheck workflows

Benefits:
- simplify API client implementation with predictable response structure
- reduce response formatting boilerplate
- improve observability and traceability of API responses
*/
package jsendx

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/tecnickcom/gogen/pkg/healthcheck"
	"github.com/tecnickcom/gogen/pkg/httpserver"
	"github.com/tecnickcom/gogen/pkg/httputil"
)

const (
	// okMessage is the default message for successful responses.
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
	Data any `json:"data"`
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
func Wrap(statusCode int, info *AppInfo, data any) *Response {
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

// JSXResp holds the configuration for the HTTP response methods.
type JSXResp struct {
	httpResp *httputil.HTTPResp
}

// NewJSXResp returns a new Response object.
func NewJSXResp(h *httputil.HTTPResp) *JSXResp {
	if h == nil {
		h = httputil.NewHTTPResp(slog.Default())
	}

	return &JSXResp{
		httpResp: h,
	}
}

// Send sends a JSON respoonse wrapped in a JSendX container.
func (jr *JSXResp) Send(ctx context.Context, w http.ResponseWriter, statusCode int, info *AppInfo, data any) {
	jr.httpResp.SendJSON(ctx, w, statusCode, Wrap(statusCode, info, data))
}

// DefaultNotFoundHandlerFunc http handler called when no matching route is found.
func (jr *JSXResp) DefaultNotFoundHandlerFunc(info *AppInfo) http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			jr.Send(r.Context(), w, http.StatusNotFound, info, "invalid endpoint")
		},
	)
}

// DefaultMethodNotAllowedHandlerFunc http handler called when a request cannot be routed.
func (jr *JSXResp) DefaultMethodNotAllowedHandlerFunc(info *AppInfo) http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			jr.Send(r.Context(), w, http.StatusMethodNotAllowed, info, "the request cannot be routed")
		},
	)
}

// DefaultPanicHandlerFunc http handler to handle panics recovered from http handlers.
func (jr *JSXResp) DefaultPanicHandlerFunc(info *AppInfo) http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			jr.Send(r.Context(), w, http.StatusInternalServerError, info, "internal error")
		},
	)
}

// DefaultIndexHandler returns the route index in JSendX format.
func (jr *JSXResp) DefaultIndexHandler(info *AppInfo) httpserver.IndexHandlerFunc {
	return func(routes []httpserver.Route) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			data := &httpserver.Index{Routes: routes}
			jr.Send(r.Context(), w, http.StatusOK, info, data)
		}
	}
}

// DefaultIPHandler returns the route ip in JSendX format.
func (jr *JSXResp) DefaultIPHandler(info *AppInfo, fn httpserver.GetPublicIPFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusOK

		ip, err := fn(r.Context())
		if err != nil {
			status = http.StatusFailedDependency
		}

		jr.Send(r.Context(), w, status, info, ip)
	}
}

// DefaultPingHandler returns a ping request in JSendX format.
func (jr *JSXResp) DefaultPingHandler(info *AppInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jr.Send(r.Context(), w, http.StatusOK, info, okMessage)
	}
}

// DefaultStatusHandler returns the server status in JSendX format.
func (jr *JSXResp) DefaultStatusHandler(info *AppInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jr.Send(r.Context(), w, http.StatusOK, info, okMessage)
	}
}

// HealthCheckResultWriter returns a new healthcheck result writer.
func (jr *JSXResp) HealthCheckResultWriter(info *AppInfo) healthcheck.ResultWriter {
	return func(ctx context.Context, w http.ResponseWriter, statusCode int, data any) {
		jr.Send(ctx, w, statusCode, info, data)
	}
}
