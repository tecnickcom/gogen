// Package logging provides an interface to configure and use the logging framework
// in a consistent way across multiple applications.
package logging

import (
	"context"
	"fmt"
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogFatal calls the default fatal logger.
// nolint:gochecknoglobals
var LogFatal = zap.L().Fatal

type ctxKey struct{}

// Syncer is an interface to allow the testing of log syncing.
type Syncer interface {
	Sync() error
}

// NewDefaultLogger configures a logger with the default fields.
func NewDefaultLogger(name, version, release, format, level string) (*zap.Logger, error) {
	l, err := NewLogger(
		WithFields(
			zap.String("program", name),
			zap.String("version", version),
			zap.String("release", release),
		),
		WithFormatStr(format),
		WithLevelStr(level),
	)
	if err != nil {
		return nil, fmt.Errorf("failed configuring default logger: %w", err)
	}

	return l, nil
}

// NewLogger configures a root logger for the application.
func NewLogger(opts ...Option) (*zap.Logger, error) {
	cfg := defaultConfig()

	for _, applyOpt := range opts {
		if err := applyOpt(cfg); err != nil {
			return nil, err
		}
	}

	var (
		disableCaller bool
		encoding      string
		levelEncoder  zapcore.LevelEncoder
		timeEncoder   zapcore.TimeEncoder
	)

	switch cfg.format {
	case noFormat:
		// no-op
	case ConsoleFormat:
		disableCaller = true
		encoding = "console"
		levelEncoder = zapcore.CapitalColorLevelEncoder
		timeEncoder = zapcore.RFC3339TimeEncoder
	case JSONFormat:
		disableCaller = true
		encoding = "json"
		levelEncoder = zapcore.LowercaseLevelEncoder
		timeEncoder = zapcore.EpochNanosTimeEncoder
	default:
		return nil, fmt.Errorf("invalid log format")
	}

	hostname, err := os.Hostname()
	if err == nil {
		hostname = ""
	}

	zapCfg := zap.Config{
		Level:    zap.NewAtomicLevelAt(cfg.level),
		Encoding: encoding,
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:   "msg",
			LevelKey:     "level",
			EncodeLevel:  levelEncoder,
			TimeKey:      "timestamp",
			EncodeTime:   timeEncoder,
			CallerKey:    "caller",
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
		OutputPaths:      cfg.outputPaths,
		ErrorOutputPaths: cfg.errorOutputPaths,
		DisableCaller:    disableCaller,
		InitialFields: map[string]interface{}{
			"hostname": hostname,
		},
	}

	l, err := zapCfg.Build()
	if err == nil {
		l = l.With(cfg.fields...)
		l = WithLevelFunctionHook(l, cfg.incMetricLogLevel)
	}
	// nolint:wrapcheck
	return l, err
}

// NopLogger returns a no operation logger.
func NopLogger() *zap.Logger {
	return zap.NewNop()
}

// Sync flushes the given logger and ignores the error.
func Sync(s Syncer) {
	// it is fine to ignore the error as we are syncing the log and adding more logs would not help
	_ = s.Sync()
}

// WithComponent creates a child logger with an extra "component" tag.
func WithComponent(ctx context.Context, comp string) *zap.Logger {
	return FromContext(ctx).With(zap.String("component", comp))
}

// WithComponentAndMethod creates a child logger with extra "component" and "method" tags.
func WithComponentAndMethod(ctx context.Context, comp, method string) *zap.Logger {
	return FromContext(ctx).With(
		zap.String("component", comp),
		zap.String("method", method),
	)
}

// FromContext retrieves a logger instance form the given context.
func FromContext(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok {
		return l
	}

	return zap.NewNop()
}

// WithLogger returns a new context with the given logger.
func WithLogger(ctx context.Context, l *zap.Logger) context.Context {
	if lp, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok && lp == l {
		return ctx // do not overwrite the same logger
	}

	return context.WithValue(ctx, ctxKey{}, l)
}

// WithLevelFunctionHook registers a function with a level string argument
// which will be called each time the Logger writes out an Entry.
func WithLevelFunctionHook(l *zap.Logger, fn IncrementLogMetricsFunc) *zap.Logger {
	fnHook := func(entry zapcore.Entry) error {
		fn(entry.Level.String())
		return nil
	}

	l = l.WithOptions(zap.Hooks(fnHook))

	// replace global logger with the configured root logger
	zap.ReplaceGlobals(l)

	return l
}

// Close closes an object and logs an error in case of failure.
func Close(ctx context.Context, obj io.Closer, errorMessage string) {
	if err := obj.Close(); err != nil {
		FromContext(ctx).Error(errorMessage, zap.Error(err))
	}
}
