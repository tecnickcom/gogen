package httputil

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"go.uber.org/zap"
)

const (
	// MimeApplicationJSON contains the mime type string for JSON content.
	MimeApplicationJSON = "application/json; charset=utf-8"

	// MimeTextPlain contains the mime type string for text content.
	MimeTextPlain = "text/plain; charset=utf-8"
)

// JSend status codes.
const (
	StatusSuccess = "success"
	StatusFail    = "fail"
	StatusError   = "error"
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
	return json.Marshal(s)
}

// SendStatus sends write a HTTP status code to the response.
func SendStatus(ctx context.Context, w http.ResponseWriter, statusCode int) {
	defer logResponse(ctx, statusCode, "")

	http.Error(w, http.StatusText(statusCode), statusCode)
}

// SendJSON sends a JSON object to the response.
func SendJSON(ctx context.Context, w http.ResponseWriter, statusCode int, data interface{}) {
	defer logResponse(ctx, statusCode, data)

	writeHeaders(w, statusCode, MimeApplicationJSON)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		logging.FromContext(ctx).Error("httputil.SendJSON()", zap.Error(err))
	}
}

// SendText sends a JSON marshaled object to the response.
func SendText(ctx context.Context, w http.ResponseWriter, statusCode int, data string) {
	defer logResponse(ctx, statusCode, data)

	writeHeaders(w, statusCode, MimeTextPlain)

	if _, err := w.Write([]byte(data)); err != nil {
		logging.FromContext(ctx).Error("httputil.SendText()", zap.Error(err))
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
func logResponse(ctx context.Context, statusCode int, data interface{}) {
	l := logging.FromContext(ctx)
	reqLog := l.With(
		zap.Int("response_code", statusCode),
		zap.String("response_message", http.StatusText(statusCode)),
		zap.Any("response_status", Status(statusCode)),
		zap.Any("response_data", data),
	)

	if statusCode >= http.StatusBadRequest { // 400+
		reqLog.Error("Request")
	} else {
		reqLog.Debug("Request")
	}
}
