package logutil

import (
	"context"
	"log/slog"
)

// HookFunc is used to intercept the log message before passing it to the underlying handler.
type HookFunc func(level LogLevel, message string)

// SlogHookHandler is a slog.Handler that wraps another handler to add custom logic.
type SlogHookHandler struct {
	slog.Handler

	hookFn HookFunc
}

// NewSlogHookHandler wraps an slog.Handler with a hook function invoked for each log record.
// A nil h falls back to the handler of the current slog.Default, captured now, so the returned
// handler never panics on first use. A nil f is tolerated too (see Handle).
func NewSlogHookHandler(h slog.Handler, f HookFunc) *SlogHookHandler {
	if h == nil {
		h = slog.Default().Handler()
	}

	return &SlogHookHandler{
		Handler: h,
		hookFn:  f,
	}
}

// Handle intercepts the log record, invokes the hook (when set), then passes the
// record to the underlying handler. A nil hook is tolerated so a handler built
// with NewSlogHookHandler(h, nil) does not panic.
func (h SlogHookHandler) Handle(ctx context.Context, record slog.Record) error {
	if h.hookFn != nil {
		h.hookFn(record.Level, record.Message)
	}

	return h.Handler.Handle(ctx, record) //nolint:wrapcheck
}

// WithAttrs returns a new SlogHookHandler whose underlying handler carries the given attributes,
// preserving the hook function so it keeps firing for derived loggers. Per the slog.Handler
// contract, an empty attribute list returns the receiver unchanged.
func (h SlogHookHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return &h
	}

	return &SlogHookHandler{
		Handler: h.Handler.WithAttrs(attrs),
		hookFn:  h.hookFn,
	}
}

// WithGroup returns a new SlogHookHandler whose underlying handler opens the given group,
// preserving the hook function so it keeps firing for derived loggers. Per the slog.Handler
// contract, an empty group name returns the receiver unchanged.
func (h SlogHookHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return &h
	}

	return &SlogHookHandler{
		Handler: h.Handler.WithGroup(name),
		hookFn:  h.hookFn,
	}
}
