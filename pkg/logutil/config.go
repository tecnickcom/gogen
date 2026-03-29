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
	}
}

// DefaultTraceIDFn returns an empty trace ID string.
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
	opt := &slog.HandlerOptions{
		Level: c.Level,
	}

	var h slog.Handler

	switch c.Format {
	case FormatJSON:
		h = slog.NewJSONHandler(c.Out, opt)
	case FormatConsole:
		h = slog.NewTextHandler(c.Out, opt)
	case FormatNone:
		h = slog.DiscardHandler
	default:
		h = slog.NewJSONHandler(os.Stderr, nil)
	}

	h = h.WithAttrs(c.CommonAttr)

	if c.HookFn != nil {
		h = NewSlogHookHandler(h, c.HookFn)
	}

	return h
}
