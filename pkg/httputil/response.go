package httputil

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"log/slog"
	"net/http"
	"time"
)

const (
	// MimeApplicationJSON contains the mime type string for JSON content.
	MimeApplicationJSON = "application/json; charset=utf-8"

	// MimeApplicationXML contains the mime type string for XML content.
	MimeApplicationXML = "application/xml; charset=utf-8"

	// MimeTextPlain contains the mime type string for text content.
	MimeTextPlain = "text/plain; charset=utf-8"
)

// XMLHeader is a default XML Declaration header suitable for use with the SendXML function.
const XMLHeader = xml.Header

// JSend status codes.
const (
	StatusSuccess = "success"
	StatusFail    = "fail"
	StatusError   = "error"
)

// log keys for response logging.
const (
	logKeyResponseDataText   = "response_txt"
	logKeyResponseDataObject = "response_data"
)

// Status translates the HTTP status code to a JSend status string.
type Status int

// MarshalJSON implements the custom marshaling function for the json encoder.
func (sc Status) MarshalJSON() ([]byte, error) {
	s := StatusSuccess

	if sc >= http.StatusBadRequest { // 400+
		s = StatusFail
	}

	if sc >= http.StatusInternalServerError { // 500+
		s = StatusError
	}

	return json.Marshal(s) //nolint:wrapcheck
}

// HTTPResp holds the configuration for the HTTP response methods.
type HTTPResp struct {
	logger *slog.Logger
}

// NewHTTPResp returns a new Response object.
func NewHTTPResp(l *slog.Logger) *HTTPResp {
	if l == nil {
		l = slog.Default()
	}

	return &HTTPResp{
		logger: l,
	}
}

// SendStatus sends write a HTTP status code to the response.
func (hr *HTTPResp) SendStatus(ctx context.Context, w http.ResponseWriter, statusCode int) {
	defer hr.logResponse(ctx, statusCode, logKeyResponseDataText, "")

	http.Error(w, http.StatusText(statusCode), statusCode)
}

// SendText sends text to the response.
func (hr *HTTPResp) SendText(ctx context.Context, w http.ResponseWriter, statusCode int, data string) {
	defer hr.logResponse(ctx, statusCode, logKeyResponseDataText, data)

	writeHeaders(w, statusCode, MimeTextPlain)

	_, err := w.Write([]byte(data))
	if err != nil {
		hr.logger.With(slog.Any("error", err)).Error("httputil.SendText()")
	}
}

// SendJSON sends a JSON object to the response.
func (hr *HTTPResp) SendJSON(ctx context.Context, w http.ResponseWriter, statusCode int, data any) {
	defer hr.logResponse(ctx, statusCode, logKeyResponseDataObject, data)

	writeHeaders(w, statusCode, MimeApplicationJSON)

	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		hr.logger.With(slog.Any("error", err)).Error("httputil.SendJSON()")
	}
}

// SendXML sends an XML object to the response.
func (hr *HTTPResp) SendXML(ctx context.Context, w http.ResponseWriter, statusCode int, xmlHeader string, data any) {
	defer hr.logResponse(ctx, statusCode, logKeyResponseDataObject, data)

	writeHeaders(w, statusCode, MimeApplicationXML)

	_, err := w.Write([]byte(xmlHeader))
	if err != nil {
		hr.logger.With(slog.Any("error", err)).Error("httputil.SendXML() unable to send XML Declaration Header")
	}

	err = xml.NewEncoder(w).Encode(data)
	if err != nil {
		hr.logger.With(slog.Any("error", err)).Error("httputil.SendXML()")
	}
}

// writeHeaders sets the content type with disabled caching.
func writeHeaders(w http.ResponseWriter, statusCode int, contentType string) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set(HeaderContentType, contentType)
	w.WriteHeader(statusCode)
}

// logResponse logs the response.
func (hr *HTTPResp) logResponse(ctx context.Context, statusCode int, dataKey string, data any) {
	resTime := time.Now().UTC()

	reqTime, ok := GetRequestTimeFromContext(ctx)
	if !ok {
		reqTime = resTime
	}

	resLog := hr.logger.With(
		slog.Int("response_code", statusCode),
		slog.String("response_message", http.StatusText(statusCode)),
		slog.Any("response_status", Status(statusCode)),
		slog.Time("response_time", resTime),
		slog.Duration("response_duration", resTime.Sub(reqTime)),
		slog.Any(dataKey, data),
	)

	if statusCode >= http.StatusBadRequest { // 400+
		resLog.Error("Response")
	} else {
		resLog.Debug("Response")
	}
}
