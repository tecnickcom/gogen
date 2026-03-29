package timeutil

import (
	"encoding/json"
	"fmt"
	"time"
)

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
	return time.Time(d).Format((*new(T)).Format())
}

// MarshalJSON encodes the date as a JSON string using type parameter T's format.
func (d DateTime[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String()) //nolint:wrapcheck
}

// UnmarshalJSON parses a JSON date string according to type parameter T's format, using UTC as the timezone.
func (d *DateTime[T]) UnmarshalJSON(data []byte) error {
	var str string

	err := json.Unmarshal(data, &str)
	if err != nil {
		return err //nolint:wrapcheck
	}

	parsed, err := time.ParseInLocation((*new(T)).Format(), str, time.UTC)
	if err != nil {
		return fmt.Errorf("unable to parse the time %s : %w", str, err)
	}

	*d = DateTime[T](parsed)

	return nil
}
