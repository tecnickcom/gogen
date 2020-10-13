package bootstrap

import (
	"context"

	"go.uber.org/zap"
)

// Option is a type alias for a function that configures the application logger
type Option func(*config)

// WithContext overrides the application context (useful for testing)
func WithContext(ctx context.Context) Option {
	return func(cfg *config) {
		cfg.context = ctx
	}
}

// WithLogger overrides the default application logger
func WithLogger(l *zap.Logger) Option {
	return func(cfg *config) {
		cfg.createLoggerFunc = func() (*zap.Logger, error) {
			return l, nil
		}
	}
}

// WithCreateLoggerFunc overrides the root logger creation function (useful for testing)
func WithCreateLoggerFunc(fn CreateLoggerFunc) Option {
	return func(cfg *config) {
		cfg.createLoggerFunc = fn
	}
}

// WithCreateMetricRegisterFunc overrides the default metrics register (useful for testing)
func WithCreateMetricRegisterFunc(fn CreateMetricRegisterFunc) Option {
	return func(cfg *config) {
		cfg.createMetricRegisterFunc = fn
	}
}
