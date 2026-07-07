package timeutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"
)

// ErrInvalidDuration is returned when a JSON value cannot be interpreted as a duration
// (unparseable string, unparseable number, or an unsupported JSON type).
var ErrInvalidDuration = errors.New("timeutil: invalid duration")

// ErrDurationOverflow is returned when a numeric JSON value is outside the int64 nanosecond range.
var ErrDurationOverflow = errors.New("timeutil: duration out of int64 range")

// Duration aliases time.Duration for JSON marshaling as human-readable strings (e.g., "1h30m") instead of nanoseconds.
//
//nolint:recvcheck
type Duration time.Duration

// String returns the duration as a human-readable string (e.g., "1h30m0s") per time.Duration.String().
func (d Duration) String() string {
	return time.Duration(d).String()
}

// MarshalJSON encodes the duration as a human-readable string (e.g., "20s", "1h") rather than a raw nanosecond integer.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String()) //nolint:wrapcheck
}

// MarshalText encodes the duration as a human-readable string. Implementing
// [encoding.TextMarshaler] lets Duration be used as a JSON map key and with
// text-based encoders (YAML, TOML, [flag.TextVar], SQL text columns).
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// UnmarshalJSON parses the duration from either a human-readable string (e.g., "20s", "1h") or a numeric nanosecond value.
// Numeric values are decoded as int64 first, so integer nanosecond durations beyond float64 precision (>= 2^53) are preserved exactly.
// Fractional numbers are truncated toward zero, and numbers outside the int64 range return [ErrDurationOverflow].
// A JSON null is a no-op, leaving the value unchanged, matching the convention of the standard library.
func (d *Duration) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	var v any

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	err := dec.Decode(&v)
	if err != nil {
		return err //nolint:wrapcheck
	}

	switch value := v.(type) {
	case json.Number:
		return d.fromNumber(value)
	case string:
		return d.fromString(value)
	default:
		return fmt.Errorf("%w: unsupported JSON type %T", ErrInvalidDuration, value)
	}
}

// UnmarshalText parses a human-readable duration string (e.g. "20s", "1h30m") via [time.ParseDuration].
// Unlike [Duration.UnmarshalJSON] it does not accept a numeric nanosecond value.
func (d *Duration) UnmarshalText(data []byte) error {
	return d.fromString(string(data))
}

// fromString decodes a human-readable duration string (e.g. "20s", "1h30m") via [time.ParseDuration].
func (d *Duration) fromString(value string) error {
	aux, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("%w: string %q: %w", ErrInvalidDuration, value, err)
	}

	*d = Duration(aux)

	return nil
}

// fromNumber decodes a JSON number as nanoseconds. Integers are decoded as int64 to
// preserve precision beyond 2^53; other values fall back to float64, are truncated
// toward zero, and are rejected with [ErrDurationOverflow] when outside the int64 range.
func (d *Duration) fromNumber(value json.Number) error {
	ns, err := value.Int64()
	if err == nil {
		*d = Duration(ns)
		return nil
	}

	aux, err := value.Float64()
	if err != nil {
		return fmt.Errorf("%w: numeric value %q: %w", ErrInvalidDuration, value.String(), err)
	}

	if aux < math.MinInt64 || aux >= math.MaxInt64 {
		return fmt.Errorf("%w: %q", ErrDurationOverflow, value.String())
	}

	*d = Duration(aux)

	return nil
}
