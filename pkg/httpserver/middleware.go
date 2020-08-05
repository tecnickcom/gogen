package httpserver

import (
	"net/http"

	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"github.com/nexmoinc/gosrvlib/pkg/requestid"
	"github.com/nexmoinc/gosrvlib/pkg/uid"
	"go.uber.org/zap"
)

// requestInjectHandler wraps all incoming requests and injects a logger in the request scoped context
func requestInjectHandler(rootLogger *zap.Logger, next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		reqID := requestid.FromHTTPRequest(r, uid.NewID128())

		reqLog := rootLogger.With(
			zap.String("request_id", reqID),
			zap.String("request_method", r.Method),
			zap.String("request_path", r.URL.Path),
			zap.String("request_query", r.URL.RawQuery),
			zap.String("request_uri", r.RequestURI),
			zap.String("request_useragent", r.UserAgent()),
			zap.String("remote_ip", r.RemoteAddr),
		)

		ctx = logging.WithLogger(ctx, reqLog)
		ctx = requestid.WithRequestID(ctx, reqID)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}
