package logutil

import (
	"errors"
	"io"
	"reflect"
)

// Option configures a [Config] instance.
type Option func(*Config) error

// WithOutWriter overrides the output destination for log messages.
// A nil writer is rejected so misconfiguration fails at construction instead of
// panicking on the first log write. A typed nil (a nil *os.File, say, held in a
// non-nil io.Writer interface) is rejected too: it panics just the same.
func WithOutWriter(w io.Writer) Option {
	return func(cfg *Config) error {
		if isNilWriter(w) {
			return errors.New("nil output writer")
		}

		cfg.Out = w

		return nil
	}
}

// isNilWriter reports whether w is nil or a typed nil: a non-nil interface holding a nil pointer, map,
// slice, func or channel.
//
// An untyped nil always panics when written to, and so does a nil pointer whose Write method has a
// pointer receiver that dereferences it, by far the common shape (a nil *os.File, *bytes.Buffer,
// *bufio.Writer). The other nilable kinds are rejected on the same suspicion rather than on proof: a
// nil map or slice with a value receiver can have a perfectly working Write. Rejecting them costs a
// caller nothing but an explicit non-nil value, whereas accepting a writer that panics on the first
// log line defeats the point of validating at construction.
func isNilWriter(w io.Writer) bool {
	if w == nil {
		return true
	}

	switch v := reflect.ValueOf(w); v.Kind() { //nolint:exhaustive // only nilable kinds can hold a typed nil.
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return v.IsNil()
	default:
		return false
	}
}

// WithFormat overrides the log output format (JSON, console, or discard).
func WithFormat(f LogFormat) Option {
	return func(cfg *Config) error {
		if !ValidFormat(f) {
			return errors.New("invalid log format")
		}

		cfg.Format = f

		return nil
	}
}

// WithFormatStr overrides the log format using a string ("json", "console", "none").
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

// WithLevel overrides the minimum log level to emit.
func WithLevel(l LogLevel) Option {
	return func(cfg *Config) error {
		if !ValidLevel(l) {
			return errors.New("invalid log level")
		}

		cfg.Level = l

		return nil
	}
}

// WithLevelStr overrides the log level using a string (e.g., "error", "debug", "trace").
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

// WithCommonAttr sets the attributes attached to every log record, replacing any
// previously configured common attributes.
func WithCommonAttr(a ...Attr) Option {
	return func(cfg *Config) error {
		cfg.CommonAttr = a
		return nil
	}
}

// WithHookFn adds a callback that intercepts each log record before the underlying handler processes it.
func WithHookFn(f HookFunc) Option {
	return func(cfg *Config) error {
		cfg.HookFn = f
		return nil
	}
}

// WithTraceIDFn adds a callback that dynamically retrieves the trace ID for each record.
func WithTraceIDFn(f TraceIDFunc) Option {
	return func(cfg *Config) error {
		cfg.TraceIDFn = f
		return nil
	}
}

// WithSource enables or disables source location (file:line) annotation on each record.
// It is disabled by default to avoid the per-record runtime.CallersFrames cost.
func WithSource(enabled bool) Option {
	return func(cfg *Config) error {
		cfg.Source = enabled
		return nil
	}
}
