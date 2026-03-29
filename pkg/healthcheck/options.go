package healthcheck

import (
	"log/slog"
)

// HandlerOption configures a [Handler] instance.
type HandlerOption func(h *Handler)

// WithResultWriter overrides how healthcheck results are serialized to HTTP output.
//
// Use this to integrate custom response envelopes or content types.
func WithResultWriter(w ResultWriter) HandlerOption {
	return func(h *Handler) {
		h.writeResult = w
	}
}

// WithLogger sets the logger used by the handler and its default result writer.
func WithLogger(logger *slog.Logger) HandlerOption {
	return func(h *Handler) {
		h.logger = logger
	}
}
