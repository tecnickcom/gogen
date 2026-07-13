package logutil

import (
	"context"
	"log/slog"
)

// TraceIDKey is the record attribute key used to carry the trace ID. It is the
// single source of truth for the field name across nurago's logging packages.
const TraceIDKey = "trace_id"

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
	inner       slog.Handler // downstream handler with the user's attrs/groups applied
	root        slog.Handler // downstream handler before any user attrs/groups
	ops         []traceOp    // user WithAttrs/WithGroup operations, in order
	grouped     bool         // whether any group is currently open
	traceAttr   bool         // whether a root-level trace ID is already carried (suppresses the injected one)
	traceGroup  bool         // whether the root group is named TraceIDKey (suppresses it only when that group renders)
	traceFilled bool         // whether attributes were added under that root group, so it is sure to render
	traceIDFn   TraceIDFunc
}

// NewSlogTraceIDHandler wraps h so each record gains a trace ID attribute resolved,
// per record, from f. A nil f returns h unchanged (no trace ID attribute is added),
// mirroring how a nil TraceIDFunc is treated elsewhere in the package. Resolving the
// trace ID per record (rather than once at construction) lets a dynamic TraceIDFunc
// reflect the current request/context on every line. The trace ID is emitted at the
// root of the record even when the logger is derived with WithGroup.
//
// A nil h falls back to the handler of the current slog.Default, captured now, so the returned
// handler never panics on first use.
//
// The result is wrapped in the sanitizing handler (see slogSanitizeHandler), so records and derived
// attributes are stripped of the groups that render nothing before the trace ID is injected: that
// injected attribute is exactly the one the standard library's elided-group separator bug would leave
// without its comma. Attributes already applied to h before it is wrapped do not pass through the
// filter, and are not covered.
//
// Neither is a slog.HandlerOptions.ReplaceAttr callback installed on h: it runs below this handler and
// can empty a group the filter has already passed by deleting its members (returning the zero slog.Attr),
// which re-creates the separator bug and can strip the trace ID. Use Config.SlogHandler, whose only
// callback never deletes, or supply a handler whose ReplaceAttr does not delete attributes.
//
// And the filter repairs a time.Time that slog's JSON encoder cannot write (a year outside [0,9999],
// which it renders as an "!ERROR:" string followed by the value, making the line invalid) only where it
// is an *attribute*. A record's own timestamp is not an attribute and never reaches the filter; it is
// repaired by the ReplaceAttr callback Config.SlogHandler installs (see replaceLevelName), which this
// constructor cannot install on a handler it merely wraps. A hand-built record carrying such a
// timestamp — slog.Logger always stamps time.Now(), so only a middleware, tee or replay handler can
// produce one — therefore still yields an invalid line here. Use Config.SlogHandler, or give h a
// ReplaceAttr that rewrites slog.TimeKey.
//
// A caller-supplied root-level trace ID wins over the injected one (see Handle). Attributes
// already applied to h before it is wrapped are invisible to that check too: apply them through the
// returned handler's WithAttrs, or use Config.SlogHandler, which accounts for Config.CommonAttr.
func NewSlogTraceIDHandler(h slog.Handler, f TraceIDFunc) slog.Handler {
	if h == nil {
		h = slog.Default().Handler()
	}

	if f == nil {
		return h
	}

	return newSlogSanitizeHandler(newSlogTraceIDHandler(h, f, false))
}

// newSlogTraceIDHandler wraps an slog.Handler so each record gains a trace ID attribute resolved
// from the given TraceIDFunc. traceAttr seeds the deduplication for attributes already baked into h
// (Config.CommonAttr, which is preformatted into h once rather than replayed per record): when it
// carries a root-level trace ID, the handler injects none of its own.
func newSlogTraceIDHandler(h slog.Handler, f TraceIDFunc, traceAttr bool) *slogTraceIDHandler {
	return &slogTraceIDHandler{
		inner:     h,
		root:      h,
		traceAttr: traceAttr,
		traceIDFn: f,
	}
}

// Enabled reports whether a record at the given level should be handled. Groups and
// attributes do not affect enablement, so it delegates to the built handler.
func (h *slogTraceIDHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle injects the resolved trace ID and passes the record to the underlying handler,
// keeping the trace ID at the root of the output, ahead of the record's own attributes.
//
// If a root-level trace ID is already present — logged as a record attribute (the reserved
// TraceIDKey, including inside an inlined empty-key group), or carried by the handler because it was
// supplied via WithAttrs/With or Config.CommonAttr (h.traceAttr), or because the root group is
// itself named TraceIDKey and renders (h.traceGroup) — the caller's value wins and no second one is
// injected, avoiding a duplicate JSON key that a last-wins parser would resolve to the injected
// value. In that case TraceIDFn is not invoked at all. Suppression requires the caller's field to
// actually render: one that is elided leaves the injected value in place, so a record never ends up
// with no trace ID at all. A trace ID supplied under an open group nests inside that group and does
// not suppress the root one.
//
// Note: the grouped branch rebuilds the downstream handler chain per record (to keep the
// trace ID at the root); the common, ungrouped case takes the fast path.
func (h *slogTraceIDHandler) Handle(ctx context.Context, record slog.Record) error {
	// Fast path: with no open group, the trace ID is prepended to a fresh record, which lands it at
	// the root. It is written ahead of the record's own attributes rather than appended to them
	// because the record handed in may be shared with other handlers (a tee, or a middleware that
	// already added attributes of its own), and mutating it would violate slog.Record's copy-sharing
	// contract — the standard library detects that and writes a "!BUG" field into the line. TraceIDFn
	// is resolved only when the value is actually injected, so a caller-supplied trace ID skips it.
	//
	// A group that renders nothing, followed by the injected attribute, would trip a separator bug in
	// the standard library's handlers; the sanitizing handler this one sits under (see
	// slogSanitizeHandler) has already removed such groups from the record, so nothing is filtered
	// here — and a record reaching this handler by another route is not protected.
	if !h.grouped {
		if h.traceAttr || (record.NumAttrs() > 0 && recordHasAttr(record, TraceIDKey)) {
			return h.inner.Handle(ctx, record) //nolint:wrapcheck
		}

		return h.inner.Handle(ctx, leadRecord(record, slog.String(TraceIDKey, h.traceIDFn()))) //nolint:wrapcheck
	}

	// A group is open: replay the user's operations from the pre-group root with the trace ID
	// injected first, so it stays at the root instead of nesting in the group. A trace ID already
	// carried at the root (h.traceAttr) is replayed with the ops, so the injected one is skipped
	// rather than duplicating it — as is a root group named TraceIDKey, but only once it is known to
	// render: whether it does depends on this record, so unlike the rest it cannot be decided at
	// derivation time, and deciding it there would drop the trace ID from every record that leaves
	// the group empty.
	target := h.root
	if !h.carriesRootTraceID(record) {
		target = target.WithAttrs([]slog.Attr{slog.String(TraceIDKey, h.traceIDFn())})
	}

	for _, op := range h.ops {
		if op.group != "" {
			target = target.WithGroup(op.group)
		} else {
			target = target.WithAttrs(op.attrs)
		}
	}

	return target.Handle(ctx, record) //nolint:wrapcheck
}

// leadAttrsInline is the number of a record's own attributes leadRecord gathers without allocating.
// slog.Record stores its first five attributes inline and spills the rest into a slice it grows
// geometrically from empty, so a record's attributes must be handed to AddAttrs in one call (see
// leadRecord) — which means gathering them first. Sixteen covers the widest request-log line in
// practice while keeping the array small enough to stay on the stack.
const leadAttrsInline = 16

// leadRecord returns a copy of record carrying lead ahead of its own attributes.
//
// The record is rebuilt rather than mutated: slog.Record's contract forbids modifying a record whose
// copies have been handed out — AddAttrs on a shared record makes the standard library write a
// "!BUG" field into the line — and a handler cannot know whether an upstream one kept a copy.
//
// The record's own attributes are gathered and added in a single AddAttrs call. Adding them one at a
// time instead makes AddAttrs regrow its overflow slice from scratch on every call (1, 2, 4, 8 …), so
// a record with more attributes than slog.Record's five inline slots — of which the injected trace ID
// takes one — costs an allocation per attribute rather than one for the whole record.
func leadRecord(record slog.Record, lead slog.Attr) slog.Record {
	var stack [leadAttrsInline]slog.Attr

	attrs := stack[:0]

	record.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)

		return true
	})

	out := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	out.AddAttrs(lead)
	out.AddAttrs(attrs...)

	return out
}

// recordHasAttr reports whether record carries an attribute with the given key at the root of the
// output. It stops at the first match and does not allocate.
func recordHasAttr(record slog.Record, key string) bool {
	found := false

	record.Attrs(func(a slog.Attr) bool {
		if attrHasRootKey(a, key) {
			found = true
			return false
		}

		return true
	})

	return found
}

// hasRootKey reports whether attrs carry the given key at the root of the output.
func hasRootKey(attrs []Attr, key string) bool {
	for _, a := range attrs {
		if attrHasRootKey(a, key) {
			return true
		}
	}

	return false
}

// attrHasRootKey reports whether a writes the given (non-empty) key at the root of the output. It
// descends into inlined (empty-key) groups, whose members slog flattens onto the enclosing level, but
// not into named groups, whose members nest under their own key. A value that writes no field (see
// attrRenders) does not count: it cannot stand in for the injected trace ID.
//
// It resolves only where the key could be carried — an attribute already under it, or an empty-key one,
// which is the only kind that can inline. That resolution costs nothing anyway: the sanitizing handler
// above has already resolved every value and substituted the result, so resolving here is a no-op, as
// it is for the standard library handler below when it writes them. Every LogValuer is invoked exactly
// once per record.
func attrHasRootKey(a Attr, key string) bool {
	// Anything under another (non-empty) key neither matches nor inlines: decided without resolving.
	if a.Key != key && a.Key != "" {
		return false
	}

	v := a.Value.Resolve()

	if a.Key == key {
		return valueRenders(v)
	}

	// An empty key: only a group inlines onto the enclosing level.
	if v.Kind() != slog.KindGroup {
		return false
	}

	for _, ga := range v.Group() {
		if attrHasRootKey(ga, key) {
			return true
		}
	}

	return false
}

// WithAttrs returns a new handler whose underlying handler carries the given attributes,
// preserving the trace ID injection. An empty attribute list returns the receiver.
//
// A trace ID among the attributes suppresses the injected one (see Handle), unless a group is
// open, in which case the attributes nest inside it and leave the root untouched.
func (h *slogTraceIDHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	return &slogTraceIDHandler{
		inner:     h.inner.WithAttrs(attrs),
		root:      h.root,
		ops:       appendTraceOp(h.ops, traceOp{attrs: attrs}),
		grouped:   h.grouped,
		traceAttr: h.traceAttr || (!h.grouped && hasRootKey(attrs, TraceIDKey)),
		// Attributes added under an open group nest inside it, so a rendering one guarantees that a
		// root group named TraceIDKey renders too, whatever the record carries.
		traceGroup:  h.traceGroup,
		traceFilled: h.traceFilled || (h.traceGroup && attrsRender(attrs)),
		traceIDFn:   h.traceIDFn,
	}
}

// WithGroup returns a new handler whose underlying handler opens the given group,
// preserving the trace ID injection at the root. An empty group name returns the receiver.
//
// A group opened at the root under the reserved TraceIDKey writes a root-level trace_id field of its
// own (an object holding the grouped attributes), so it suppresses the injected one rather than
// colliding with it — but only for records that give it something to hold, since slog elides a group
// that renders nothing. That is decided per record in Handle, so a record which leaves the group
// empty still carries the injected trace ID rather than an empty object in its place.
//
// The suppression only ever removes the *injected* trace ID. A caller who supplies the key twice
// themselves — With(slog.String(TraceIDKey, ...)) and then WithGroup(TraceIDKey) — gets both of their
// own fields at the root, as they would from a bare standard-library handler: one is a preformatted
// attribute and the other an open group, and dropping either would discard what they asked to log.
func (h *slogTraceIDHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	return &slogTraceIDHandler{
		inner:       h.inner.WithGroup(name),
		root:        h.root,
		ops:         appendTraceOp(h.ops, traceOp{group: name}),
		grouped:     true,
		traceAttr:   h.traceAttr,
		traceGroup:  h.traceGroup || (!h.grouped && name == TraceIDKey),
		traceFilled: h.traceFilled,
		traceIDFn:   h.traceIDFn,
	}
}

// carriesRootTraceID reports whether the handler already puts a root-level trace ID on this record,
// so the injected one would duplicate it: one baked in via WithAttrs/With or Config.CommonAttr, or a
// root group named TraceIDKey that renders — which depends on the record, since slog elides a group
// that holds nothing, and an elided one supplies no trace ID.
func (h *slogTraceIDHandler) carriesRootTraceID(record slog.Record) bool {
	if h.traceAttr {
		return true
	}

	return h.traceGroup && (h.traceFilled || recordRenders(record))
}

// appendTraceOp returns a new slice with op appended, without aliasing the input slice.
func appendTraceOp(ops []traceOp, op traceOp) []traceOp {
	out := make([]traceOp, len(ops)+1)
	copy(out, ops)
	out[len(ops)] = op

	return out
}
