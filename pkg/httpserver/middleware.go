package httpserver

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"time"

	libhttputil "github.com/tecnickcom/gogen/pkg/httputil"
	"github.com/tecnickcom/gogen/pkg/traceid"
	"github.com/tecnickcom/gogen/pkg/uidc"
)

// MiddlewareArgs contains extra optional arguments to be passed to the middleware handler function MiddlewareFn.
type MiddlewareArgs struct {
	// Method is the HTTP method (e.g.: GET, POST, PUT, DELETE, ...).
	Method string

	// Path is the URL path.
	Path string

	// Description is the description of the route or a general description for the handler.
	Description string

	// TraceIDHeaderName is the Trace ID header name.
	TraceIDHeaderName string

	// RedactFunc is the function used to redact HTTP request and response dumps in the logs.
	RedactFunc RedactFn

	// Logger is the logger.
	Logger *slog.Logger
}

// MiddlewareFn is a function that wraps an http.Handler.
type MiddlewareFn func(args MiddlewareArgs, next http.Handler) http.Handler

// RequestInjectHandler wraps all incoming requests and injects a logger in the request scoped context.
func RequestInjectHandler(logger *slog.Logger, traceIDHeaderName string, redactFn RedactFn, next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		reqTime := time.Now().UTC()
		reqID := traceid.FromHTTPRequestHeader(r, traceIDHeaderName, uidc.NewID128())

		ctx := r.Context()
		ctx = libhttputil.WithRequestTime(ctx, reqTime)
		ctx = traceid.NewContext(ctx, reqID)

		logger = logger.With(
			slog.String(traceid.DefaultLogKey, reqID),
			slog.Time("request_time", reqTime),
			slog.String("request_method", r.Method),
			slog.String("request_path", r.URL.Path),
			slog.String("request_query", r.URL.RawQuery),
			slog.String("request_remote_address", r.RemoteAddr),
			slog.String("request_uri", r.RequestURI),
			slog.String("request_user_agent", r.UserAgent()),
			slog.String("request_x_forwarded_for", r.Header.Get("X-Forwarded-For")),
		)

		dbglog := logger.Enabled(ctx, slog.LevelDebug)

		if dbglog {
			reqDump, _ := httputil.DumpRequest(r, true)
			logger = logger.With(slog.String("request_dump", redactFn(string(reqDump))))
		}

		next.ServeHTTP(w, r.WithContext(ctx))

		if dbglog {
			logger.Debug("request")
			return
		}

		logger.Info("request")
	}

	return http.HandlerFunc(fn)
}

// LoggerMiddlewareFn returns the middleware handler function to handle logs.
func LoggerMiddlewareFn(args MiddlewareArgs, next http.Handler) http.Handler {
	return RequestInjectHandler(args.Logger, args.TraceIDHeaderName, args.RedactFunc, next)
}

// ApplyMiddleware returns an http Handler with all middleware handler functions applied.
func ApplyMiddleware(arg MiddlewareArgs, next http.Handler, middleware ...MiddlewareFn) http.Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		next = middleware[i](arg, next)
	}

	return next
}
