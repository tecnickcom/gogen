/*
Package slogx provides a generic log/slog compatible interface for structured logging capabilities.
*/
package slogx

// Logger defines the minimal interface used by this package for structured logging.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

// NewNop returns a no-op Logger.
func NewNop() Logger {
	return &NopLogger{}
}

// NopLogger is a no-op implementation of the Logger interface.
type NopLogger struct{}

// Debug is a no-op.
func (*NopLogger) Debug(_ string, _ ...any) {}

// Info is a no-op.
func (*NopLogger) Info(_ string, _ ...any) {}

// Warn is a no-op.
func (*NopLogger) Warn(_ string, _ ...any) {}

// Error is a no-op.
func (*NopLogger) Error(_ string, _ ...any) {}

// With is a no-op.
func (l *NopLogger) With(_ ...any) Logger {
	return l
}
