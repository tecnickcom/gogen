package timeutil

import (
	"encoding/json"
	"fmt"
	"time"
)

// DateTimeType is an interface that defines a method to return a time format string.
type DateTimeType interface {
	Format() string
}

// DateTime is a generic type that wraps time.Time and provides JSON marshaling/unmarshaling
// using the specified time format defined by the type parameter T.
type DateTime[T DateTimeType] time.Time

// Time returns the underlying time.Time value.
func (d DateTime[T]) Time() time.Time {
	return time.Time(d)
}

// String returns a string representing the date in the format returned by T.Format().
func (d DateTime[T]) String() string {
	return time.Time(d).Format((*new(T)).Format())
}

// MarshalJSON implements the json.Marshaler interface.
func (d DateTime[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String()) //nolint:wrapcheck
}

// UnmarshalJSON implements the json.Unmarshaler interface.
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
