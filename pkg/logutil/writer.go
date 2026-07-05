package logutil

import (
	"context"
	"log/slog"
)

// SlogWriter is a custom io.Writer that writes to slog.Logger at a configurable level.
type SlogWriter struct {
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

	w.Logger.Log(context.Background(), w.Level, msg)

	return len(p), nil
}
