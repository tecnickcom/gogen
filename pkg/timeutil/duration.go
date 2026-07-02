package timeutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

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

// UnmarshalJSON parses the duration from either a human-readable string (e.g., "20s", "1h") or a numeric nanosecond value.
// Numeric values are decoded as int64 first, so integer nanosecond durations beyond float64 precision (>= 2^53) are preserved exactly.
func (d *Duration) UnmarshalJSON(data []byte) error {
	var v any

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	err := dec.Decode(&v)
	if err != nil {
		return err //nolint:wrapcheck
	}

	switch value := v.(type) {
	case json.Number:
		ns, err := value.Int64()
		if err == nil {
			*d = Duration(ns)
			return nil
		}

		aux, err := value.Float64()
		if err != nil {
			return fmt.Errorf("unable to parse the numeric time duration %s : %w", value, err)
		}

		*d = Duration(aux)

		return nil
	case string:
		aux, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("unable to parse the time duration %s : %w", value, err)
		}

		*d = Duration(aux)

		return nil
	default:
		return fmt.Errorf("invalid time duration type: %v", value)
	}
}
