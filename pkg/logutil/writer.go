package logutil

import (
	"log/slog"
)

// SlogWriter is a custom io.Writer that writes to slog.Logger at the error level.
type SlogWriter struct {
	Logger *slog.Logger
}

// NewSlogWriter constructs an io.Writer that routes writes to an slog.Logger at error level.
func NewSlogWriter(logger *slog.Logger) *SlogWriter {
	return &SlogWriter{Logger: logger}
}

// Write logs the input bytes at error level, stripping trailing newlines, and returns bytes written.
func (w SlogWriter) Write(p []byte) (int, error) {
	msg := string(p)

	// Remove trailing newline character from the log message.
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}

	w.Logger.Error(msg)

	return len(p), nil
}
