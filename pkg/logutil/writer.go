package logutil

import (
	"context"
	"log/slog"
)

// SlogWriter is a custom io.Writer that writes to slog.Logger at a configurable level.
//
// The zero value is usable: it writes to slog.Default at LevelInfo.
type SlogWriter struct {
	// Logger is the destination. A nil Logger writes to slog.Default, resolved per write, so the
	// zero value works and a logger installed later is picked up.
	Logger *slog.Logger
	// Level is the severity used for every bridged line. The zero value is
	// LevelInfo; construct with NewSlogWriter for the Error default.
	Level LogLevel
}

// NewSlogWriter constructs an io.Writer that routes writes to an slog.Logger at error level.
func NewSlogWriter(logger *slog.Logger) *SlogWriter {
	return NewSlogWriterLevel(logger, LevelError)
}

// NewSlogWriterLevel constructs an io.Writer that routes writes to an slog.Logger at the
// given level. This is the level-aware counterpart to NewSlogWriter: bridged standard
// log.Logger output is not necessarily error-severity, so callers can pick the level that
// matches the source. A nil logger falls back to slog.Default so writes never panic.
func NewSlogWriterLevel(logger *slog.Logger, level LogLevel) *SlogWriter {
	if logger == nil {
		logger = slog.Default()
	}

	return &SlogWriter{Logger: logger, Level: level}
}

// Write logs the input bytes at the configured level, stripping trailing newlines, and returns bytes written.
func (w SlogWriter) Write(p []byte) (int, error) {
	msg := string(p)

	// Remove trailing newline character from the log message.
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}

	w.logger().Log(context.Background(), w.Level, msg)

	return len(p), nil
}

// logger returns the destination logger, falling back to slog.Default for the zero value (see
// SlogWriter), so a write never panics on a nil Logger.
func (w SlogWriter) logger() *slog.Logger {
	if w.Logger == nil {
		return slog.Default()
	}

	return w.Logger
}
