package logutil

import (
	"io"
	"log/slog"
	"os"
)

// Attr is a type alias for slog.Attr.
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

// DefaultConfig returns a Config instance with default settings.
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

// NewConfig returns a new configuration with the applied options.
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

// SlogDefaultLogger set and return a slog logger based on the Config settings.
func (c *Config) SlogDefaultLogger() *slog.Logger {
	l := c.SlogLogger()

	slog.SetDefault(l)

	return l
}

// SlogLogger returns a slog logger based on the Config settings.
func (c *Config) SlogLogger() *slog.Logger {
	return slog.New(c.SlogHandler())
}

// SlogHandler returns a new slog Handler based on the Config settings.
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
