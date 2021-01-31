// Package jsendx implements a custom JSEND model to wrap HTTP responses in a JSON object with default fields.
package jsendx

import (
	"context"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/nexmoinc/gosrvlib/pkg/healthcheck"
	"github.com/nexmoinc/gosrvlib/pkg/httpserver"
	"github.com/nexmoinc/gosrvlib/pkg/httpserver/route"
	"github.com/nexmoinc/gosrvlib/pkg/httputil"
	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"go.uber.org/zap"
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

// NewRouter create a new router configured to responds with JSend wrapper responses for 404, 405 and panic.
func NewRouter(info *AppInfo, instrumentHandler httpserver.InstrumentHandler) *httprouter.Router {
	r := httprouter.New()

	r.NotFound = instrumentHandler("404", func(w http.ResponseWriter, r *http.Request) {
		Send(r.Context(), w, http.StatusNotFound, info, "invalid endpoint")
	})

	r.MethodNotAllowed = instrumentHandler("405", func(w http.ResponseWriter, r *http.Request) {
		Send(r.Context(), w, http.StatusMethodNotAllowed, info, "the request cannot be routed")
	})

	r.PanicHandler = func(w http.ResponseWriter, r *http.Request, p interface{}) {
		logging.FromContext(r.Context()).Error("panic",
			zap.Any("err", p),
			zap.String("stacktrace", string(debug.Stack())),
		)
		Send(r.Context(), w, http.StatusInternalServerError, info, "internal error")
	}

	return r
}

// DefaultIndexHandler returns the route index in JSendX format.
func DefaultIndexHandler(info *AppInfo) httpserver.IndexHandlerFunc {
	return func(routes []route.Route) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			data := &route.Index{Routes: routes}
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
