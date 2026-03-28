/*
Package healthcheck provides a simple reusable framework for defining and
collecting health checks for external services or application components.

It solves the problem of consolidating liveness and readiness logic by
standardizing health check definitions and offering concurrent result
collection.

The package is intended to be used with an HTTP handler that aggregates checks
and returns a combined health status.

Top features:
- declarative HealthChecker interface for pluggable checks
- lightweight health check registration and configuration
- concurrency-friendly result collection for faster health probes

Benefits:
- make service health monitoring consistent across components
- reduce boilerplate when wiring health check endpoints
- simplify integration with orchestrators and monitoring tools

For an implementation example, see examples/service/internal/cli/bind.go.
*/
package healthcheck

import (
	"context"
)

// HealthChecker is the interface that wraps the HealthCheck method.
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// HealthCheck is a structure containing the configuration for a single health check.
type HealthCheck struct {
	// ID is a unique identifier for the healthcheck.
	ID string

	// Checker is the function used to perform the healthchecks.
	Checker HealthChecker
}

// New creates a new instance of a health check configuration with default timeout.
func New(id string, checker HealthChecker) HealthCheck {
	return HealthCheck{
		ID:      id,
		Checker: checker,
	}
}
