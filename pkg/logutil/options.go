package logutil

import (
	"errors"
	"io"
)

// Option is a type alias for a function that updates the Config.
type Option func(*Config) error

// WithOutWriter overrides the output io.Writer.
func WithOutWriter(w io.Writer) Option {
	return func(cfg *Config) error {
		cfg.Out = w
		return nil
	}
}

// WithFormat overrides the log format.
func WithFormat(f LogFormat) Option {
	return func(cfg *Config) error {
		if !ValidFormat(f) {
			return errors.New("invalid log format")
		}

		cfg.Format = f

		return nil
	}
}

// WithFormatStr overrides the log format.
func WithFormatStr(f string) Option {
	return func(cfg *Config) error {
		lf, err := ParseFormat(f)
		if err != nil {
			return err
		}

		cfg.Format = lf

		return nil
	}
}

// WithLevel overrides the log level.
func WithLevel(l LogLevel) Option {
	return func(cfg *Config) error {
		if !ValidLevel(l) {
			return errors.New("invalid log format")
		}

		cfg.Level = l

		return nil
	}
}

// WithLevelStr overrides the log level.
func WithLevelStr(l string) Option {
	return func(cfg *Config) error {
		ll, err := ParseLevel(l)
		if err != nil {
			return err
		}

		cfg.Level = ll

		return nil
	}
}

// WithCommonAttr adds common attributes to the logger.
func WithCommonAttr(a ...Attr) Option {
	return func(cfg *Config) error {
		cfg.CommonAttr = a
		return nil
	}
}

func WithLevelHookFn(f LevelHookFunc) Option {
	return func(cfg *Config) error {
		cfg.LevelHookFn = f
		return nil
	}
}
