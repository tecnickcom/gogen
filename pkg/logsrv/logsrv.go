/*
Package logsrv provides a high-performance zerolog backend exposed through the
standard log/slog API.

# Problem

Teams often want the ecosystem compatibility of slog while still leveraging
zerolog's speed and compact structured output. Without an adapter layer,
applications end up with mixed logging APIs, inconsistent severity mapping, and
duplicated setup logic across services.

# Solution

This package bridges [log/slog] and zerolog with a native slog.Handler that
writes each record's attributes directly onto a zerolog Event. It reuses the
shared configuration model from gogen's logutil package.

[NewLogger] creates a slog.Logger backed by zerolog and applies:
  - log format selection (JSON, console, discard),
  - common structured attributes,
  - trace ID injection,
  - optional hook execution,
  - and full syslog level names (via logutil.LevelName).

# Compatibility

The logging model is compatible with:
  - Nicola Asuni, 2014-08-11, "Software Logging Format",
    https://technick.net/guides/software/software_logging_format/

See also:
  - github.com/tecnickcom/gogen/pkg/logutil

# Notes

Fields are written in the order the attributes were added (record attributes follow
the common/WithAttrs attributes), not sorted. Duration and time values are rendered
using zerolog's process-global TimeFieldFormat and DurationFieldUnit settings.

The emitted "level" field carries the full syslog severity name via logutil.LevelName
("emergency", "alert", "critical", "error", "warning", "notice", "info", "debug",
"trace"), matching logutil's backend — the extended severities are not collapsed onto
zerolog's fixed level set. In FormatConsole mode, zerolog's ConsoleWriter colorizes only
its own level vocabulary, so the extended names render without color. Errors logged at the
"error" or "err" key are rendered as their message string. A nil value (slog.Any(key, nil))
is emitted as a null field.

The built-in field names "level", "time", "message", "source" and "trace_id" are reserved:
a user attribute with the same name is written in addition to the built-in one, producing a
duplicate JSON key (as in the standard library's slog handlers). The writer (cfg.Out) is
wrapped so concurrent logging is safe even for a non-thread-safe destination.

# Benefits

logsrv lets applications keep the standard slog interface while using zerolog's
performance characteristics and structured logging ergonomics.
*/
package logsrv

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime"

	"github.com/rs/zerolog"
	"github.com/tecnickcom/gogen/pkg/logutil"
)

// NewLogger constructs a slog.Logger backed by zerolog, configured via logutil.Config,
// and installs it as the process-wide slog default.
//
// Use [NewHandler] (for example slog.New(logsrv.NewHandler(cfg))) when a logger is
// needed without replacing the global default.
//
// A nil cfg falls back to logutil.DefaultConfig. See [NewHandler] for the details of
// format selection, attributes, trace-ID injection, hooks, and level naming.
func NewLogger(cfg *logutil.Config) *slog.Logger {
	sl := slog.New(NewHandler(cfg))

	slog.SetDefault(sl)

	return sl
}

// NewHandler constructs the slog.Handler backing a logsrv logger, without mutating any
// global logger state. Applies format selection, common attributes, trace-ID injection,
// hooks, and full syslog level naming. A nil cfg falls back to logutil.DefaultConfig, and a
// nil Out writer falls back to os.Stderr, so construction never yields a handler that panics
// on the first write.
//
// The trace ID is resolved per record via cfg.TraceIDFn (matching the logutil model), so
// a dynamic TraceIDFn reflects the current request/context on every line rather than being
// frozen at construction. The handler writes it natively at the root of every record — even
// for loggers derived with WithGroup — and a caller-supplied root trace_id takes precedence.
// A nil TraceIDFn is valid and simply omits the trace ID field.
//
// The hook (cfg.HookFn) is invoked at the slog layer, before the record is handed to
// zerolog, so it receives the original record level (e.g. logutil.LevelNotice or
// logutil.LevelCritical) rather than any derived value.
//
// Note: the emitted "level" field carries the full syslog severity name via logutil.LevelName
// (e.g. "critical", "notice", "emergency"), matching logutil's backend rather than collapsing
// onto zerolog's fixed level set. In FormatConsole mode, zerolog's ConsoleWriter colorizes only
// its own level vocabulary, so the extended names render (uncolored) as their upper-cased prefix.
func NewHandler(cfg *logutil.Config) slog.Handler {
	if cfg == nil {
		cfg = logutil.DefaultConfig()
	}

	// FormatNone with no hook has nothing to write and no side effect to fire, so a
	// zero-cost DiscardHandler (Enabled == false) is used instead of running the full
	// zerolog encode path into io.Discard on every record.
	if cfg.Format == logutil.FormatNone && cfg.HookFn == nil {
		return slog.DiscardHandler
	}

	out := cfg.Out
	if out == nil {
		out = os.Stderr
	}

	// SyncWriter serializes writes so concurrent logging to a non-thread-safe cfg.Out (or the
	// stateful ConsoleWriter) is race-free, matching logutil's standard-library backend. os.Stderr
	// is already serialized at the runtime FD layer; the extra lock is negligible.
	//
	// TraceLevel base so zerolog never gates a record itself: enablement is decided solely at the
	// slog layer (via Enabled), matching cfg.Level exactly — including the Trace level, which a
	// default (Debug-level) zerolog logger would otherwise drop.
	zl := zerolog.New(zerolog.SyncWriter(writerByFormat(cfg.Format, out))).Level(zerolog.TraceLevel)

	var h slog.Handler = &zerologHandler{
		logger:    zl,
		traceIDFn: cfg.TraceIDFn,
		minLevel:  cfg.Level,
		source:    cfg.Source,
	}

	// Bake the common attributes into the zerolog context once, so they are serialized
	// a single time and memcpy'd per record rather than re-encoded on every line.
	h = h.WithAttrs(cfg.CommonAttr)

	// The trace ID is written natively by the handler (at the root of every record, even under an
	// open group), so no trace-ID wrapper is needed. Wrap with the hook handler (as logutil does)
	// instead of hooking the zerolog event: a zerolog hook would only see the NoLevel event,
	// losing the original severity.
	if cfg.HookFn != nil {
		h = logutil.NewSlogHookHandler(h, cfg.HookFn)
	}

	return h
}

// groupFrame is one open WithGroup level together with the WithAttrs added within it.
// Attributes added under an open group must nest below the group in the output, so they
// cannot be baked into the zerolog context (which only appends at the root); they are
// replayed per record instead.
type groupFrame struct {
	name  string
	attrs []slog.Attr
}

// zerologHandler is the native leaf slog.Handler. It writes records directly onto zerolog
// Events. Pre-group attributes are baked into logger via zerolog's context so the common,
// ungrouped path allocates nothing; open groups fall back to per-record nesting.
type zerologHandler struct {
	logger    zerolog.Logger      // pre-group attributes baked in via With()
	groups    []groupFrame        // open groups (nil on the common ungrouped fast path)
	traceIDFn logutil.TraceIDFunc // resolves the root trace ID per record (nil omits the field)
	minLevel  slog.Level          // minimum enabled level (from logutil.Config.Level)
	source    bool                // whether to emit the caller "source" field
}

// Enabled reports whether a record at the given level should be handled. Groups and
// attributes do not affect enablement.
func (h *zerologHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.minLevel
}

// Handle writes the record onto a zerolog Event: level, timestamp, optional source, the
// root trace ID, the record's attributes (nested under any open groups), and the message.
func (h *zerologHandler) Handle(_ context.Context, record slog.Record) error {
	// NoLevel so zerolog writes no level field of its own; the level is written below as the
	// full syslog name, preserving the extended severities instead of collapsing them onto
	// zerolog's fixed set.
	e := h.logger.WithLevel(zerolog.NoLevel)

	e.Str(zerolog.LevelFieldName, logutil.LevelName(record.Level))

	// Omit the time field for a zero timestamp, matching slog's own handlers.
	if !record.Time.IsZero() {
		e.Time(zerolog.TimestampFieldName, record.Time)
	}

	if h.source && record.PC != 0 {
		e.Dict("source", sourceDict(record))
	}

	if len(h.groups) == 0 {
		h.writeRoot(e, &record)

		e.Msg(record.Message)

		return nil
	}

	// A group is open: the trace ID stays at the root while the record's attributes nest in the
	// group, which is omitted entirely when it (and its subgroups) carry no fields so output
	// matches slog's empty-group elision.
	if h.traceIDFn != nil {
		e.Str(logutil.TraceIDKey, h.traceIDFn())
	}

	if dict, wrote := h.buildGroupDict(h.groups, &record); wrote {
		e.Dict(h.groups[0].name, dict)
	}

	e.Msg(record.Message)

	return nil
}

// WithAttrs returns a handler carrying the given attributes. With no open group the
// attributes are baked into the zerolog context (keeping the fast path allocation-free);
// under an open group they are appended to the innermost frame for per-record nesting.
// Per the slog.Handler contract, an empty attribute list returns the receiver.
func (h *zerologHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	nh := *h

	if len(h.groups) == 0 {
		c := h.logger.With()
		for _, a := range attrs {
			c = applyContext(c, a)
		}

		nh.logger = c.Logger()

		return &nh
	}

	nh.groups = make([]groupFrame, len(h.groups))
	copy(nh.groups, h.groups)
	last := len(nh.groups) - 1
	merged := make([]slog.Attr, 0, len(nh.groups[last].attrs)+len(attrs))
	merged = append(merged, nh.groups[last].attrs...)
	merged = append(merged, attrs...)
	nh.groups[last].attrs = merged

	return &nh
}

// WithGroup returns a handler that nests subsequent attributes under name. Per the
// slog.Handler contract, an empty group name returns the receiver unchanged.
func (h *zerologHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	nh := *h
	nh.groups = make([]groupFrame, len(h.groups)+1)
	copy(nh.groups, h.groups)
	nh.groups[len(h.groups)] = groupFrame{name: name}

	return &nh
}

// writeRoot writes the record's attributes at the root (the ungrouped fast path) and injects the
// trace ID unless the caller already supplied a root-level trace_id, in which case theirs wins and
// TraceIDFn is not invoked — matching logutil's trace handler.
func (h *zerologHandler) writeRoot(e *zerolog.Event, record *slog.Record) {
	hasTrace := false

	record.Attrs(func(a slog.Attr) bool {
		if a.Key == logutil.TraceIDKey {
			hasTrace = true
		}

		addAttrEvent(e, a)

		return true
	})

	if h.traceIDFn != nil && !hasTrace {
		e.Str(logutil.TraceIDKey, h.traceIDFn())
	}
}

// buildGroupDict builds the nested zerolog dictionary for the open group chain, adding each
// frame's attributes and, at the innermost frame, the record's attributes. It reports whether
// the chain produced any field, so the caller can omit an all-empty (sub)group rather than
// render it as a bare "{}" object. This is a single pass — attribute values are resolved once.
func (h *zerologHandler) buildGroupDict(frames []groupFrame, record *slog.Record) (*zerolog.Event, bool) {
	d := zerolog.Dict()
	wrote := false

	for _, a := range frames[0].attrs {
		if addAttrEvent(d, a) {
			wrote = true
		}
	}

	if len(frames) > 1 {
		if child, childWrote := h.buildGroupDict(frames[1:], record); childWrote {
			d.Dict(frames[1].name, child)

			wrote = true
		}
	} else {
		record.Attrs(func(a slog.Attr) bool {
			if addAttrEvent(d, a) {
				wrote = true
			}

			return true
		})
	}

	if !wrote {
		// Recycle the unused pooled event rather than leaking it: Send on a detached dict
		// (nil writer) emits nothing and returns the event to zerolog's pool.
		d.Send()

		return nil, false
	}

	return d, wrote
}

// addAttrEvent writes a single slog.Attr onto a zerolog Event (which may be the record's root
// event or a group Dict) and reports whether it produced a field. The attribute value is
// resolved exactly once. The zero Attr is elided, and a group that yields no fields is elided
// (its enclosing key is not emitted).
//
//nolint:gocyclo,cyclop // exhaustive slog.Kind dispatch needs one case per kind.
func addAttrEvent(e *zerolog.Event, a slog.Attr) bool {
	if a.Equal(slog.Attr{}) {
		return false
	}

	v := a.Value.Resolve()
	wrote := true

	switch v.Kind() {
	case slog.KindGroup:
		wrote = addGroupEvent(e, a.Key, v.Group())
	case slog.KindString:
		e.Str(a.Key, v.String())
	case slog.KindInt64:
		e.Int64(a.Key, v.Int64())
	case slog.KindUint64:
		e.Uint64(a.Key, v.Uint64())
	case slog.KindFloat64:
		e.Float64(a.Key, v.Float64())
	case slog.KindBool:
		e.Bool(a.Key, v.Bool())
	case slog.KindDuration:
		e.Dur(a.Key, v.Duration())
	case slog.KindTime:
		e.Time(a.Key, v.Time())
	case slog.KindAny, slog.KindLogValuer:
		addAnyEvent(e, a.Key, v)
	}

	return wrote
}

// addGroupEvent writes a group's attributes onto e and reports whether it produced any field:
// inlined at the current level when the key is empty, otherwise nested under a sub-dictionary
// keyed by name. An empty group emits nothing (no bare "{}").
func addGroupEvent(e *zerolog.Event, key string, attrs []slog.Attr) bool {
	if key == "" {
		wrote := false

		for _, ga := range attrs {
			if addAttrEvent(e, ga) {
				wrote = true
			}
		}

		return wrote
	}

	d := zerolog.Dict()
	wrote := false

	for _, ga := range attrs {
		if addAttrEvent(d, ga) {
			wrote = true
		}
	}

	if !wrote {
		d.Send() // recycle the unused pooled event (see buildGroupDict)

		return false
	}

	e.Dict(key, d)

	return true
}

// addAnyEvent writes an arbitrary value onto e, rendering an error as its message string
// (zerolog's native error form) and any other value via reflection.
func addAnyEvent(e *zerolog.Event, key string, v slog.Value) {
	x := v.Any()
	if err, ok := x.(error); ok {
		e.AnErr(key, err)

		return
	}

	e.Interface(key, x)
}

// applyContext bakes a single slog.Attr into a zerolog context (used to precompute the
// common/WithAttrs attributes at the root). It mirrors addAttrEvent's type handling.
//
//nolint:gocyclo,cyclop // exhaustive slog.Kind dispatch needs one case per kind.
func applyContext(c zerolog.Context, a slog.Attr) zerolog.Context {
	if a.Equal(slog.Attr{}) {
		return c
	}

	v := a.Value.Resolve()

	switch v.Kind() {
	case slog.KindGroup:
		c = applyGroupContext(c, a.Key, v.Group())
	case slog.KindString:
		c = c.Str(a.Key, v.String())
	case slog.KindInt64:
		c = c.Int64(a.Key, v.Int64())
	case slog.KindUint64:
		c = c.Uint64(a.Key, v.Uint64())
	case slog.KindFloat64:
		c = c.Float64(a.Key, v.Float64())
	case slog.KindBool:
		c = c.Bool(a.Key, v.Bool())
	case slog.KindDuration:
		c = c.Dur(a.Key, v.Duration())
	case slog.KindTime:
		c = c.Time(a.Key, v.Time())
	case slog.KindAny, slog.KindLogValuer:
		c = applyAnyContext(c, a.Key, v)
	}

	return c
}

// applyGroupContext bakes a group's attributes into c: inlined at the root when the key is
// empty, otherwise nested under a sub-dictionary keyed by name. An empty group bakes nothing.
func applyGroupContext(c zerolog.Context, key string, attrs []slog.Attr) zerolog.Context {
	if key == "" {
		for _, ga := range attrs {
			c = applyContext(c, ga)
		}

		return c
	}

	d := zerolog.Dict()
	wrote := false

	for _, ga := range attrs {
		if addAttrEvent(d, ga) {
			wrote = true
		}
	}

	if !wrote {
		d.Send() // recycle the unused pooled event (see buildGroupDict)

		return c
	}

	return c.Dict(key, d)
}

// applyAnyContext bakes an arbitrary value into c, rendering an error as its message string
// and any other value via reflection.
func applyAnyContext(c zerolog.Context, key string, v slog.Value) zerolog.Context {
	x := v.Any()
	if err, ok := x.(error); ok {
		return c.AnErr(key, err)
	}

	return c.Interface(key, x)
}

// sourceDict builds the caller-location "source" object (function, file, line), mirroring
// slog's AddSource group. The caller guards against a zero PC before invoking it.
func sourceDict(record slog.Record) *zerolog.Event {
	fs := runtime.CallersFrames([]uintptr{record.PC})
	f, _ := fs.Next()

	return zerolog.Dict().
		Str("function", f.Function).
		Str("file", f.File).
		Int("line", f.Line)
}

// writerByFormat returns the zerolog output writer for the specified format (JSON, console, or discard).
func writerByFormat(f logutil.LogFormat, w io.Writer) io.Writer {
	switch f {
	case logutil.FormatJSON:
		return w
	case logutil.FormatConsole:
		// Colorize only when the destination is a terminal, so console output written
		// to a file or pipe does not embed raw ANSI escape sequences.
		return zerolog.ConsoleWriter{Out: w, NoColor: !isTerminalWriter(w)}
	case logutil.FormatNone:
		return io.Discard
	default:
		return w
	}
}

// isTerminalWriter reports whether w is a terminal (character device). Non-terminal
// writers (files, pipes, in-memory buffers) return false so console output is emitted
// without color escapes.
//
// It only recognizes a bare *os.File: a terminal wrapped in a decorator (e.g. a
// bufio.Writer) is treated as non-terminal and rendered without color. This is a
// deliberate, dependency-free heuristic (golang.org/x/term is not an allowed import);
// callers needing precise control should pass the terminal *os.File directly.
func isTerminalWriter(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}

	info, err := f.Stat()

	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
