package testutil

import (
	"context"

	"github.com/Vonage/gosrvlib/pkg/logging"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// Context returns a context initialized with a NOP logger for testing.
func Context() context.Context {
	return logging.WithLogger(context.Background(), zap.NewNop())
}

// ContextWithLogObserver returns a context initialized with a NOP logger for testing.
func ContextWithLogObserver(level zapcore.LevelEnabler) (context.Context, *observer.ObservedLogs) {
	core, logs := observer.New(level)
	l := zap.New(core)

	return logging.WithLogger(context.Background(), l), logs
}

// ContextWithHTTPRouterParams creates a context copy containing map of URL path segments.
func ContextWithHTTPRouterParams(ctx context.Context, params map[string]string) context.Context {
	var m httprouter.Params

	for k, v := range params {
		m = append(m, httprouter.Param{Key: k, Value: v})
	}

	return context.WithValue(ctx, httprouter.ParamsKey, m)
}
