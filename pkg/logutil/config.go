package logutil

import (
	"io"
	"log/slog"
	"os"
)

// Attr aliases [slog.Attr] for package-local configuration types.
type Attr = slog.Attr

// TraceIDFunc is the type of function used to retrieve a Trace ID.
type TraceIDFunc func() string

// Config holds common logger parameters.
type Config struct {
	Out        io.Writer
	Format     LogFormat
	Level      LogLevel
	CommonAttr []Attr
	HookFn     HookFunc
	TraceIDFn  TraceIDFunc
	Source     bool
}

// DefaultConfig returns a pre-initialized Config with stderr output, JSON format, info level, and empty trace ID.
func DefaultConfig() *Config {
	return &Config{
		Out:        os.Stderr,
		Format:     FormatJSON,
		Level:      LevelInfo,
		CommonAttr: []Attr{},
		HookFn:     nil,
		TraceIDFn:  defaultTraceID,
		Source:     false,
	}
}

// defaultTraceID returns an empty trace ID string. It is non-nil by design so the
// default configuration emits a stable, always-present "trace_id" field (empty
// until a real TraceIDFn is supplied via WithTraceIDFn), keeping the log schema
// consistent across records. Pass WithTraceIDFn(nil) to omit the field entirely.
func defaultTraceID() string {
	return ""
}

// NewConfig constructs a Config by applying options to DefaultConfig.
func NewConfig(opts ...Option) (*Config, error) {
	cfg := DefaultConfig()

	for _, applyOpt := range opts {
		err := applyOpt(cfg)
		if err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// SlogDefaultLogger constructs a slog.Logger from Config settings and installs it as the process default.
//
// As a side effect of slog.SetDefault, this also redirects the standard library log
// package's default output through the returned logger's handler. Use SlogLogger to
// obtain a logger without mutating that global state.
func (c *Config) SlogDefaultLogger() *slog.Logger {
	l := c.SlogLogger()

	slog.SetDefault(l)

	return l
}

// SlogLogger constructs a slog.Logger from Config settings (format, level, common attributes, hooks).
func (c *Config) SlogLogger() *slog.Logger {
	return slog.New(c.SlogHandler())
}

// OutWriter returns the effective output destination: Config.Out, or os.Stderr when Out is unusable —
// nil, or a typed nil (a nil *os.File, say, held in a non-nil io.Writer interface), which Out being an
// exported field allows even though WithOutWriter rejects it. Both backends resolve the destination
// through it, so a hand-built Config never yields a handler that panics on the first write.
func (c *Config) OutWriter() io.Writer {
	if isNilWriter(c.Out) {
		return os.Stderr
	}

	return c.Out
}

// SlogHandler constructs a slog.Handler from Config settings with optional hook interception.
func (c *Config) SlogHandler() slog.Handler {
	// FormatNone with no hook has nothing to write and no side effect to fire, so a
	// zero-cost DiscardHandler (Enabled == false) is used instead of encoding every
	// record into io.Discard.
	if c.Format == FormatNone && c.HookFn == nil {
		return slog.DiscardHandler
	}

	out := c.OutWriter()

	// ReplaceAttr renders the syslog-style level names (see replaceLevelName). Note it
	// makes slog invoke the callback for every attribute of every record, disabling some
	// of slog's precomputed-attribute fast paths; it is the cost of the extended-severity
	// labels the package models.
	opt := &slog.HandlerOptions{
		Level:       c.Level,
		AddSource:   c.Source,
		ReplaceAttr: replaceLevelName,
	}

	var h slog.Handler

	switch c.Format {
	case FormatJSON:
		h = slog.NewJSONHandler(out, opt)
	case FormatConsole:
		h = slog.NewTextHandler(out, opt)
	case FormatNone:
		// A hook is configured (the no-hook case returned above): discard output via
		// io.Discard rather than slog.DiscardHandler, whose Enabled == false would
		// silently prevent the hook (and trace) wrappers below from firing. Writing to
		// io.Discard keeps level-based enablement intact so the hook still runs.
		//nolint:sloglint // intentional: DiscardHandler would disable the hook/trace wrappers (see comment above).
		h = slog.NewJSONHandler(io.Discard, opt)
	default:
		h = slog.NewJSONHandler(out, opt)
	}

	// The common attributes are preformatted into the handler once, before the trace wrapper is
	// installed: applying them through the wrapper instead would record them as a derivation the
	// wrapper has to replay on every record of a grouped logger (to keep the trace ID at the root),
	// re-encoding them per line. The wrapper cannot see attributes already baked into h, so the
	// trace-ID deduplication for a CommonAttr-supplied trace_id is seeded explicitly here.
	//
	// They are filtered first: they are preformatted in a single WithAttrs call, so one attribute among
	// them that the standard library encodes incorrectly — a group that renders nothing, or a time whose
	// year it cannot write — would corrupt every line the handler ever writes (see slogSanitizeHandler).
	// Nothing above can filter them, since they never pass through it again.
	common, _ := sanitizeAttrs(c.CommonAttr)
	h = h.WithAttrs(common)

	if c.TraceIDFn != nil {
		h = newSlogTraceIDHandler(h, c.TraceIDFn, hasRootKey(common, TraceIDKey))
	}

	// The sanitizing handler goes above the trace wrapper, not below it: every record and every
	// derivation then reaches the wrapper already stripped of the groups that render nothing, so the
	// trace ID it injects can never be the attribute that follows one — and the per-record replay a
	// grouped logger performs runs on a chain the sanitizer is not part of, keeping it off the hot
	// path. It is installed whatever else is configured, so a nil TraceIDFn (which leaves no trace
	// wrapper at all) is protected just the same.
	h = newSlogSanitizeHandler(h)

	if c.HookFn != nil {
		h = NewSlogHookHandler(h, c.HookFn)
	}

	return h
}
