package httpserver

import (
	"net/http"

	"github.com/nexmoinc/gosrvlib/pkg/httputil"
	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"github.com/nexmoinc/gosrvlib/pkg/uid"
	"go.uber.org/zap"
)

const (
	headerRequestID = "x-request-id"
)

// requestInjectHandler wraps all incoming requests and injects a logger in the request scoped context
func requestInjectHandler(rootLogger *zap.Logger, next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		reqID := httputil.HeaderOrDefault(r, headerRequestID, uid.NewID128())

		reqLog := rootLogger.With(
			zap.String("request_id", reqID),
			zap.String("request_method", r.Method),
			zap.String("request_path", r.URL.Path),
			// zap.Any("request_query", r.URL.Query()),
			zap.String("request_query", r.URL.RawQuery),
			zap.String("request_uri", r.RequestURI),
			zap.String("request_useragent", r.UserAgent()),
			zap.String("remote_ip", r.RemoteAddr),
		)

		reqCtx := logging.WithLogger(r.Context(), reqLog)
		next.ServeHTTP(w, r.WithContext(reqCtx))
	}
	return http.HandlerFunc(fn)
}
