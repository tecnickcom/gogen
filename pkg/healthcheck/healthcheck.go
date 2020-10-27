// Package healthcheck provides a simple way to define health checks
// for external services or components.
// These checks will be aggregated in the /status endpoint.
package healthcheck

import (
	"context"
	"time"
)

const (
	// DefaultTimeout is the default timeout value for a single healthcheck
	DefaultTimeout = 1 * time.Second
)

// HealthChecker is the interface that wraps the HealthCheck method.
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// HealthCheck is a structure containing the configuration for a single health check
type HealthCheck struct {
	ID      string
	Checker HealthChecker
	Timeout time.Duration
}

// New creates a new instance of a health check configuration with default timeout
func New(id string, checker HealthChecker) HealthCheck {
	return HealthCheck{
		ID:      id,
		Checker: checker,
		Timeout: DefaultTimeout,
	}
}

// NewWithTimeout creates a new instance of a health check configuration
func NewWithTimeout(id string, checker HealthChecker, timeout time.Duration) HealthCheck {
	return HealthCheck{
		ID:      id,
		Checker: checker,
		Timeout: timeout,
	}
}
