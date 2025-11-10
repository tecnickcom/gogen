package logutil

import (
	"io"
	"log/slog"
	"os"
)

// Attr is a type alias for slog.Attr.
type Attr = slog.Attr

// LevelHookFunc is an adaptor to allow the use of an ordinary function as a Hook.
// The argument is the log level.
type LevelHookFunc func(string)

// Config holds common logger parameters.
type Config struct {
	Out         io.Writer
	Format      LogFormat
	Level       LogLevel
	CommonAttr  []Attr
	LevelHookFn LevelHookFunc
}

// DefaultConfig returns a Config instance with default settings.
func DefaultConfig() *Config {
	return &Config{
		Out:         os.Stderr,
		Format:      FormatJSON,
		Level:       LevelInfo,
		CommonAttr:  []slog.Attr{},
		LevelHookFn: nil,
	}
}
