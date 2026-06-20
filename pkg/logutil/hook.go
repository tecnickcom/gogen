package logutil

import (
	"context"
	"log/slog"
)

// traceIDKey is the record attribute key used to carry the trace ID.
const traceIDKey = "trace_id"

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

// WithAttrs returns a new SlogHookHandler whose underlying handler carries the given attributes,
// preserving the hook function so it keeps firing for derived loggers.
func (h SlogHookHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &SlogHookHandler{
		Handler: h.Handler.WithAttrs(attrs),
		hookFn:  h.hookFn,
	}
}

// WithGroup returns a new SlogHookHandler whose underlying handler opens the given group,
// preserving the hook function so it keeps firing for derived loggers.
func (h SlogHookHandler) WithGroup(name string) slog.Handler {
	return &SlogHookHandler{
		Handler: h.Handler.WithGroup(name),
		hookFn:  h.hookFn,
	}
}

// slogTraceIDHandler is a slog.Handler that injects a dynamically resolved trace ID
// attribute into every log record before delegating to the underlying handler.
type slogTraceIDHandler struct {
	slog.Handler

	traceIDFn TraceIDFunc
}

// newSlogTraceIDHandler wraps an slog.Handler so each record gains a trace ID attribute
// resolved from the given TraceIDFunc.
func newSlogTraceIDHandler(h slog.Handler, f TraceIDFunc) *slogTraceIDHandler {
	return &slogTraceIDHandler{
		Handler:   h,
		traceIDFn: f,
	}
}

// Handle adds the resolved trace ID attribute to the record and passes it to the underlying handler.
func (h slogTraceIDHandler) Handle(ctx context.Context, record slog.Record) error {
	record.AddAttrs(slog.String(traceIDKey, h.traceIDFn()))
	return h.Handler.Handle(ctx, record) //nolint:wrapcheck
}

// WithAttrs returns a new slogTraceIDHandler whose underlying handler carries the given attributes,
// preserving the trace ID function so it keeps injecting for derived loggers.
func (h slogTraceIDHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &slogTraceIDHandler{
		Handler:   h.Handler.WithAttrs(attrs),
		traceIDFn: h.traceIDFn,
	}
}

// WithGroup returns a new slogTraceIDHandler whose underlying handler opens the given group,
// preserving the trace ID function so it keeps injecting for derived loggers.
func (h slogTraceIDHandler) WithGroup(name string) slog.Handler {
	return &slogTraceIDHandler{
		Handler:   h.Handler.WithGroup(name),
		traceIDFn: h.traceIDFn,
	}
}
