package timeutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ErrParseDateTime is returned when a JSON string cannot be parsed using type parameter T's layout.
var ErrParseDateTime = errors.New("timeutil: unable to parse datetime")

// DateTimeType provides the layout string used by [DateTime] JSON marshaling/unmarshalling.
type DateTimeType interface {
	Format() string
}

// DateTime wraps [time.Time] and applies the format defined by T during JSON (un)marshaling.
type DateTime[T DateTimeType] time.Time

// Time returns the underlying time.Time value.
func (d DateTime[T]) Time() time.Time {
	return time.Time(d)
}

// String formats the date as a string according to type parameter T's Format() method.
func (d DateTime[T]) String() string {
	var layout T

	return time.Time(d).Format(layout.Format())
}

// MarshalJSON encodes the date as a JSON string using type parameter T's format.
func (d DateTime[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String()) //nolint:wrapcheck
}

// MarshalText encodes the date using type parameter T's format. Implementing
// [encoding.TextMarshaler] lets DateTime be used as a JSON map key and with
// text-based encoders (YAML, TOML, [flag.TextVar], SQL text columns).
func (d DateTime[T]) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// UnmarshalJSON parses a JSON date string according to type parameter T's format.
// A JSON null is a no-op, leaving the value unchanged, matching [time.Time.UnmarshalJSON].
// See [DateTime.UnmarshalText] for the timezone handling of layouts without an explicit zone.
func (d *DateTime[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	var str string

	err := json.Unmarshal(data, &str)
	if err != nil {
		return err //nolint:wrapcheck
	}

	return d.UnmarshalText([]byte(str))
}

// UnmarshalText parses a date string according to type parameter T's format. Layouts
// without an explicit timezone (e.g. [TDateOnly], [TTimeOnly], [TKitchen], the TStamp
// variants) are interpreted as UTC.
func (d *DateTime[T]) UnmarshalText(data []byte) error {
	var layout T

	text := string(data)

	parsed, err := time.ParseInLocation(layout.Format(), text, time.UTC)
	if err != nil {
		return fmt.Errorf("%w %q: %w", ErrParseDateTime, text, err)
	}

	*d = DateTime[T](parsed)

	return nil
}
