package httputil

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"time"

	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"go.uber.org/zap"
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

// SendStatus sends write a HTTP status code to the response.
func SendStatus(ctx context.Context, w http.ResponseWriter, statusCode int) {
	defer logResponse(ctx, statusCode, logKeyResponseDataText, "")

	http.Error(w, http.StatusText(statusCode), statusCode)
}

// SendText sends text to the response.
func SendText(ctx context.Context, w http.ResponseWriter, statusCode int, data string) {
	defer logResponse(ctx, statusCode, logKeyResponseDataText, data)

	writeHeaders(w, statusCode, MimeTextPlain)

	if _, err := w.Write([]byte(data)); err != nil {
		logging.FromContext(ctx).Error("httputil.SendText()", zap.Error(err))
	}
}

// SendJSON sends a JSON object to the response.
func SendJSON(ctx context.Context, w http.ResponseWriter, statusCode int, data interface{}) {
	defer logResponse(ctx, statusCode, logKeyResponseDataObject, data)

	writeHeaders(w, statusCode, MimeApplicationJSON)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		logging.FromContext(ctx).Error("httputil.SendJSON()", zap.Error(err))
	}
}

// SendXML sends an XML object to the response.
func SendXML(ctx context.Context, w http.ResponseWriter, statusCode int, xmlHeader string, data interface{}) {
	defer logResponse(ctx, statusCode, logKeyResponseDataObject, data)

	writeHeaders(w, statusCode, MimeApplicationXML)

	if _, err := w.Write([]byte(xmlHeader)); err != nil {
		logging.FromContext(ctx).Error("httputil.SendXML() unable to send XML Declaration Header", zap.Error(err))
	}

	if err := xml.NewEncoder(w).Encode(data); err != nil {
		logging.FromContext(ctx).Error("httputil.SendXML()", zap.Error(err))
	}
}

// writeHeaders sets the content type with disabled caching.
func writeHeaders(w http.ResponseWriter, statusCode int, contentType string) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(statusCode)
}

// logResponse logs the response.
func logResponse(ctx context.Context, statusCode int, dataKey string, data interface{}) {
	resTime := time.Now().UTC()

	reqTime, ok := GetRequestTimeFromContext(ctx)
	if !ok {
		reqTime = resTime
	}

	l := logging.FromContext(ctx)
	resLog := l.With(
		zap.Int("response_code", statusCode),
		zap.String("response_message", http.StatusText(statusCode)),
		zap.Any("response_status", Status(statusCode)),
		zap.Time("response_time", resTime),
		zap.Duration("response_duration", resTime.Sub(reqTime)),
		zap.Any(dataKey, data),
	)

	if statusCode >= http.StatusBadRequest { // 400+
		resLog.Error("Response")
	} else {
		resLog.Debug("Response")
	}
}
