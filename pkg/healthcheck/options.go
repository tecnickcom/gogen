package healthcheck

import (
	"log/slog"
)

// HandlerOption is a type alias for a function that configures the healthcheck HTTP handler.
type HandlerOption func(h *Handler)

// WithResultWriter overrides the default healthcheck result writer.
func WithResultWriter(w ResultWriter) HandlerOption {
	return func(h *Handler) {
		h.writeResult = w
	}
}

// WithLogger overrides the default logger.
func WithLogger(logger *slog.Logger) HandlerOption {
	return func(h *Handler) {
		h.logger = logger
	}
}
