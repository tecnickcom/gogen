package jsendx

import (
	"context"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/nexmoinc/gosrvlib/pkg/httpserver"
	"github.com/nexmoinc/gosrvlib/pkg/httpserver/route"
	"github.com/nexmoinc/gosrvlib/pkg/httputil"
	"github.com/nexmoinc/gosrvlib/pkg/metrics"
)

// Response wraps data into a JSend compliant response
type Response struct {
	Program   string          `json:"program"`   // Program name
	Version   string          `json:"version"`   // Program version
	Release   string          `json:"release"`   // Program release number
	URL       string          `json:"url"`       // Public URL of this service
	DateTime  string          `json:"datetime"`  // Human-readable date and time when the event occurred
	Timestamp int64           `json:"timestamp"` // Machine-readable UTC timestamp in nanoseconds since EPOCH
	Status    httputil.Status `json:"status"`    // Status code (error|fail|success)
	Code      int             `json:"code"`      // HTTP status code
	Message   string          `json:"message"`   // Error or status message
	Data      interface{}     `json:"data"`      // Data payload
}

// AppInfo is a struct containing data to enrich the JSend response
type AppInfo struct {
	ProgramName    string
	ProgramVersion string
	ProgramRelease string
	ServerAddress  string
}

// Wrap sends an Response object.
func Wrap(statusCode int, info *AppInfo, data interface{}) *Response {
	now := time.Now().UTC()

	r := Response{
		Program:   info.ProgramName,
		Version:   info.ProgramVersion,
		Release:   info.ProgramRelease,
		URL:       info.ServerAddress,
		DateTime:  now.Format(time.RFC3339),
		Timestamp: now.UnixNano(),
		Status:    httputil.Status(statusCode),
		Code:      statusCode,
		Message:   http.StatusText(statusCode),
		Data:      data,
	}

	return &r
}

// Send sends a JSON respoonse wrapped in a JSendX container
func Send(ctx context.Context, w http.ResponseWriter, statusCode int, info *AppInfo, data interface{}) {
	httputil.SendJSON(ctx, w, statusCode, Wrap(statusCode, info, data))
}

// NewRouter create a new router configured to responds with JSend wrapper responses for 404, 405 and panic
func NewRouter(info *AppInfo) *httprouter.Router {
	r := httprouter.New()

	r.NotFound = metrics.Handler("404", func(w http.ResponseWriter, r *http.Request) {
		Send(r.Context(), w, http.StatusNotFound, info, "invalid endpoint")
	})

	r.MethodNotAllowed = metrics.Handler("405", func(w http.ResponseWriter, r *http.Request) {
		Send(r.Context(), w, http.StatusMethodNotAllowed, info, "the request cannot be routed")
	})

	r.PanicHandler = func(w http.ResponseWriter, r *http.Request, p interface{}) {
		Send(r.Context(), w, http.StatusInternalServerError, info, "internal error")
	}

	return r
}

// DefaultStatusHandler returns the server status in JSendX format
func DefaultStatusHandler(info *AppInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		Send(r.Context(), w, http.StatusOK, info, "OK")
	}
}

// DefaultPingHandler returns a ping request in JSendX format
func DefaultPingHandler(info *AppInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		Send(r.Context(), w, http.StatusOK, info, "OK")
	}
}

// DefaultRoutesIndexHandler returns the route index in JSendX format
func DefaultRoutesIndexHandler(info *AppInfo) httpserver.RouteIndexHandlerFunc {
	return func(routes []route.Route) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			data := &route.Index{Routes: routes}
			Send(r.Context(), w, http.StatusOK, info, data)
		}
	}
}
