package logutil

import (
	"log"
	"log/slog"
)

// NewLogFromSlog creates a standard log.Logger that writes to the provided slog.Logger.
func NewLogFromSlog(logger *slog.Logger) *log.Logger {
	return log.New(NewSlogWriter(logger), "", 0)
}
