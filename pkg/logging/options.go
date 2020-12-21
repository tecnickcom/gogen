package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Option is a type alias for a function that configures the application logger
type Option func(*config) error

// WithFormat manually overrides the environment log format
func WithFormat(f Format) Option {
	return func(cfg *config) error {
		cfg.format = f
		return nil
	}
}

// WithFormatStr manually overrides the environment log format
func WithFormatStr(f string) Option {
	return func(cfg *config) error {
		lf, err := ParseFormat(f)
		if err != nil {
			return err
		}
		cfg.format = lf
		return nil
	}
}

// WithLevel manually overrides the environment log level
func WithLevel(l zapcore.Level) Option {
	return func(cfg *config) error {
		cfg.level = l
		return nil
	}
}

// WithLevelStr manually overrides the environment log level
func WithLevelStr(l string) Option {
	return func(cfg *config) error {
		ll, err := ParseLevel(l)
		if err != nil {
			return err
		}
		cfg.level = ll
		return nil
	}
}

// WithFields add static fields to the logger
func WithFields(f ...zap.Field) Option {
	return func(cfg *config) error {
		cfg.fields = f
		return nil
	}
}

// WithIncrementLogMetricsFunc replaces the default log level metrics function
func WithIncrementLogMetricsFunc(fn IncrementLogMetricsFunc) Option {
	return func(cfg *config) error {
		cfg.incMetricLogLevel = fn
		return nil
	}
}

// WithOutputPaths manually overrides the OutputPaths option
func WithOutputPaths(paths []string) Option {
	return func(cfg *config) error {
		cfg.outputPaths = paths
		return nil
	}
}

// WithErrorOutputPaths manually overrides the ErrorOutputPaths option
func WithErrorOutputPaths(paths []string) Option {
	return func(cfg *config) error {
		cfg.errorOutputPaths = paths
		return nil
	}
}
