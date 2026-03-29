package logutil

import (
	"log"
	"log/slog"
)

// NewLogFromSlog constructs a standard log.Logger that routes writes to an slog.Logger.
func NewLogFromSlog(logger *slog.Logger) *log.Logger {
	return log.New(NewSlogWriter(logger), "", 0)
}
