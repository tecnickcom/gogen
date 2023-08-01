package timeutil

import (
	"encoding/json"
	"fmt"
	"time"
)

// Duration is an alias for the standard time.Duration.
type Duration time.Duration

// String returns a string representing the duration in the form "72h3m0.5s".
// It is a wrapper for time.Duration.String().
func (d Duration) String() string {
	return time.Duration(d).String()
}

// MarshalJSON returns d as the JSON encoding of d.
// It encodes the time.Duration in human readable format (e.g.: 20s, 1h, ...).
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String()) //nolint:wrapcheck
}

// UnmarshalJSON sets *d to a copy of data.
// It converts human readable time duration format (e.g.: 20s, 1h, ...) in standard time.Duration.
func (d *Duration) UnmarshalJSON(data []byte) error {
	var v any

	if err := json.Unmarshal(data, &v); err != nil {
		return err //nolint:wrapcheck
	}

	switch value := v.(type) {
	case float64:
		*d = Duration(value)
		return nil
	case string:
		aux, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("unable to parse the time duration %s :%w", value, err)
		}

		*d = Duration(aux)

		return nil
	default:
		return fmt.Errorf("invalid time duration type: %v", value)
	}
}
