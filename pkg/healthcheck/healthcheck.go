package healthcheck

import (
	"context"
	"encoding/json"
)

// HealthChecker is the interface that wraps the HealthCheck method.
type HealthChecker interface {
	HealthCheck(ctx context.Context) Result
}

// Result represents the result of a health check
type Result struct {
	Status Status `json:"status"`
	Error  error  `json:"error,omitempty"`
}

// Status is a type representing the status of a health check
type Status uint8

const (
	// Unavailable means the service being checked was not reachable
	Unavailable Status = iota

	// OK means the service being checked is behaving as expected
	OK

	// Err means the service being checked has errors
	Err
)

func (s Status) String() string {
	switch s {
	case OK:
		return "OK"
	case Err:
		return "ERR"
	}
	return "N/A"
}

// MarshalJSON implements the custom marshaling function for the json encoder
func (s Status) MarshalJSON() (b []byte, e error) {
	return json.Marshal(s.String())
}
