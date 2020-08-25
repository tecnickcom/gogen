package healthcheck

// HandlerOption is a type alias for a function that configures the healthcheck HTTP handler
type HandlerOption func(h *Handler)

// WithResultWriter overrides the default healthcheck result writer
func WithResultWriter(w ResultWriter) HandlerOption {
	return func(h *Handler) {
		h.writeResult = w
	}
}
