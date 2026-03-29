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
func NewSlogHookHandler(h slog.Handler, f HookFunc) *SlogHookHandler {
	return &SlogHookHandler{
		Handler: h,
		hookFn:  f,
	}
}

// Handle intercepts the log record, invokes the hook, then passes the record to the underlying handler.
func (h SlogHookHandler) Handle(ctx context.Context, record slog.Record) error {
	h.hookFn(record.Level, record.Message)
	return h.Handler.Handle(ctx, record) //nolint:wrapcheck
}
