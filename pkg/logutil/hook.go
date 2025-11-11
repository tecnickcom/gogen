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

// NewSlogHookHandler adds a hook function to the slog Handler.
func NewSlogHookHandler(h slog.Handler, f HookFunc) *SlogHookHandler {
	return &SlogHookHandler{
		Handler: h,
		hookFn:  f,
	}
}

// Handle intercepts the log record, modifies the message, and then passes
// it to the underlying handler.
func (h SlogHookHandler) Handle(ctx context.Context, record slog.Record) error {
	h.hookFn(record.Level, record.Message)
	return h.Handler.Handle(ctx, record) //nolint:wrapcheck
}
