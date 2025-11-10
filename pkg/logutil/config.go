package logutil

import (
	"context"
	"io"
	"log/slog"
	"os"
)

// Attr is a type alias for slog.Attr.
type Attr = slog.Attr

// HookFunc is an adaptor to allow the use of an ordinary function as a Hook.
// The argument is the log level.

// HookFunc is used to intercept the log message before passing it to the underlying handler.
type HookFunc func(level LogLevel, message string)

// Config holds common logger parameters.
type Config struct {
	Out        io.Writer
	Format     LogFormat
	Level      LogLevel
	CommonAttr []Attr
	HookFn     HookFunc
}

// DefaultConfig returns a Config instance with default settings.
func DefaultConfig() *Config {
	return &Config{
		Out:        os.Stderr,
		Format:     FormatJSON,
		Level:      LevelInfo,
		CommonAttr: []Attr{},
		HookFn:     nil,
	}
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

	if c.HookFn != nil {
		h = &SlogHookHandler{
			Handler: h,
			hookFn:  c.HookFn,
		}
	}

	return h.WithAttrs(c.CommonAttr)
}

// SlogHookHandler is a slog.Handler that wraps another handler to add custom logic.
type SlogHookHandler struct {
	slog.Handler

	hookFn HookFunc
}

// Handle intercepts the log record, modifies the message, and then passes
// it to the underlying handler.
func (h SlogHookHandler) Handle(ctx context.Context, record slog.Record) error {
	h.hookFn(record.Level, record.Message)
	return h.Handler.Handle(ctx, record) //nolint:wrapcheck
}
