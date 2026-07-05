package logutil

import (
	"context"
	"log/slog"
)

// TraceIDKey is the record attribute key used to carry the trace ID. It is the
// single source of truth for the field name across gogen's logging packages.
const TraceIDKey = "trace_id"

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

// traceOp records one WithAttrs or WithGroup operation applied to a
// slogTraceIDHandler so the trace ID can be re-injected at the root even when a
// group is open. Exactly one of its fields is set per operation.
type traceOp struct {
	attrs []slog.Attr // set for a WithAttrs operation
	group string      // non-empty for a WithGroup operation
}

// slogTraceIDHandler is a slog.Handler that injects a dynamically resolved trace ID
// attribute into every log record, keeping it at the root of the output even for
// loggers derived with WithGroup.
type slogTraceIDHandler struct {
	inner     slog.Handler // downstream handler with the user's attrs/groups applied
	root      slog.Handler // downstream handler before any user attrs/groups
	ops       []traceOp    // user WithAttrs/WithGroup operations, in order
	grouped   bool         // whether any group is currently open
	traceIDFn TraceIDFunc
}

// NewSlogTraceIDHandler wraps h so each record gains a trace ID attribute resolved,
// per record, from f. A nil f returns h unchanged (no trace ID attribute is added),
// mirroring how a nil TraceIDFunc is treated elsewhere in the package. Resolving the
// trace ID per record (rather than once at construction) lets a dynamic TraceIDFunc
// reflect the current request/context on every line. The trace ID is emitted at the
// root of the record even when the logger is derived with WithGroup.
func NewSlogTraceIDHandler(h slog.Handler, f TraceIDFunc) slog.Handler {
	if f == nil {
		return h
	}

	return newSlogTraceIDHandler(h, f)
}

// newSlogTraceIDHandler wraps an slog.Handler so each record gains a trace ID attribute
// resolved from the given TraceIDFunc.
func newSlogTraceIDHandler(h slog.Handler, f TraceIDFunc) *slogTraceIDHandler {
	return &slogTraceIDHandler{
		inner:     h,
		root:      h,
		traceIDFn: f,
	}
}

// Enabled reports whether a record at the given level should be handled. Groups and
// attributes do not affect enablement, so it delegates to the built handler.
func (h *slogTraceIDHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle injects the resolved trace ID and passes the record to the underlying handler,
// keeping the trace ID at the root of the output.
//
// If the record already carries a root-level trace ID (a caller logged the reserved
// TraceIDKey), the caller's value wins and no second one is injected, avoiding a duplicate
// JSON key; in that case TraceIDFn is not invoked at all. TraceIDKey is reserved; supplying
// it via CommonAttr or a pre-group With is not deduplicated.
//
// Note: the grouped branch rebuilds the downstream handler chain per record (to keep the
// trace ID at the root); the common, ungrouped case takes the allocation-free fast path.
func (h *slogTraceIDHandler) Handle(ctx context.Context, record slog.Record) error {
	// Fast path: with no open group, adding the attribute to the record lands the
	// trace ID at the root. The record is created per call by slog.Logger and is not
	// shared with other handlers that retain it, so mutating it here is safe without
	// Record.Clone. TraceIDFn is resolved only when the value is actually injected, so a
	// caller-supplied trace ID short-circuits it.
	if !h.grouped {
		if record.NumAttrs() == 0 || !recordHasAttr(record, TraceIDKey) {
			record.AddAttrs(slog.String(TraceIDKey, h.traceIDFn()))
		}

		return h.inner.Handle(ctx, record) //nolint:wrapcheck
	}

	// A group is open: replay the user's operations from the pre-group root with the
	// trace ID injected first, so it stays at the root instead of nesting in the group.
	target := h.root.WithAttrs([]slog.Attr{slog.String(TraceIDKey, h.traceIDFn())})
	for _, op := range h.ops {
		if op.group != "" {
			target = target.WithGroup(op.group)
		} else {
			target = target.WithAttrs(op.attrs)
		}
	}

	return target.Handle(ctx, record) //nolint:wrapcheck
}

// recordHasAttr reports whether record carries a top-level attribute with the given key.
// It stops at the first match and does not allocate.
func recordHasAttr(record slog.Record, key string) bool {
	found := false

	record.Attrs(func(a slog.Attr) bool {
		if a.Key == key {
			found = true
			return false
		}

		return true
	})

	return found
}

// WithAttrs returns a new handler whose underlying handler carries the given attributes,
// preserving the trace ID injection. An empty attribute list returns the receiver.
func (h *slogTraceIDHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	return &slogTraceIDHandler{
		inner:     h.inner.WithAttrs(attrs),
		root:      h.root,
		ops:       appendTraceOp(h.ops, traceOp{attrs: attrs}),
		grouped:   h.grouped,
		traceIDFn: h.traceIDFn,
	}
}

// WithGroup returns a new handler whose underlying handler opens the given group,
// preserving the trace ID injection at the root. An empty group name returns the receiver.
func (h *slogTraceIDHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	return &slogTraceIDHandler{
		inner:     h.inner.WithGroup(name),
		root:      h.root,
		ops:       appendTraceOp(h.ops, traceOp{group: name}),
		grouped:   true,
		traceIDFn: h.traceIDFn,
	}
}

// appendTraceOp returns a new slice with op appended, without aliasing the input slice.
func appendTraceOp(ops []traceOp, op traceOp) []traceOp {
	out := make([]traceOp, len(ops)+1)
	copy(out, ops)
	out[len(ops)] = op

	return out
}
