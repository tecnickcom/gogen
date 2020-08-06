package testutil

import (
	"context"

	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// Context returns a context initialized with a NOP logger for testing
func Context() context.Context {
	return logging.WithLogger(context.Background(), zap.NewNop())
}

// ContextWithLogObserver returns a context initialized with a NOP logger for testing
func ContextWithLogObserver(level zapcore.Level) (context.Context, *observer.ObservedLogs) {
	core, logs := observer.New(level)
	l := zap.New(core)
	return logging.WithLogger(context.Background(), l), logs
}
