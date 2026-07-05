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

// SlogHandler constructs a slog.Handler from Config settings with optional hook interception.
func (c *Config) SlogHandler() slog.Handler {
	// FormatNone with no hook has nothing to write and no side effect to fire, so a
	// zero-cost DiscardHandler (Enabled == false) is used instead of encoding every
	// record into io.Discard.
	if c.Format == FormatNone && c.HookFn == nil {
		return slog.DiscardHandler
	}

	// Guard against a nil writer (e.g. a hand-built Config) so construction never
	// yields a handler that panics on the first write.
	out := c.Out
	if out == nil {
		out = os.Stderr
	}

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

	h = h.WithAttrs(c.CommonAttr)

	h = NewSlogTraceIDHandler(h, c.TraceIDFn)

	if c.HookFn != nil {
		h = NewSlogHookHandler(h, c.HookFn)
	}

	return h
}
