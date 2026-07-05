/*
Package healthcheck provides a reusable framework for defining, executing, and
aggregating service health probes.

# Problem

Services commonly need to report health across multiple dependencies (databases,
HTTP upstreams, queues, internal subsystems). Implementing these checks ad hoc
often leads to inconsistent response formats, duplicated handler boilerplate,
and slow serial probing.

# Solution

This package standardizes health probing around three core pieces:
  - [HealthChecker]: pluggable check contract (`HealthCheck(context.Context) error`),
    with [HealthCheckFunc] to adapt a plain function
  - [HealthCheck]: lightweight ID + checker registration unit
  - [Handler]: HTTP aggregator that runs checks concurrently and writes a
    combined result

The default handler response is JSON and maps each check ID to either "OK" or
the check error message.

# Aggregation Semantics

  - checks execute in parallel for faster probe completion
  - overall HTTP status is 200 when all checks pass
  - overall HTTP status is 503 when any check fails
  - response payload always includes per-check results
  - a checker panic is recovered, logged, and reported as a failed check
  - with [WithTimeout], checks that overrun are reported as failed

Check IDs should be unique and non-empty: results are keyed by ID, so duplicate
or empty IDs collapse into a single entry (a warning is logged at construction).

Result-writing behavior is customizable via [WithResultWriter], making it easy
to integrate with custom envelopes (for example JSendX) while keeping the
execution model unchanged.

# HTTP Probe Helper

[CheckHTTPStatus] is included as a convenience helper for external HTTP
dependencies. It supports context timeout control and request customization via
[WithConfigureRequest].

# Benefits

healthcheck makes health endpoint behavior consistent, concurrent, and easy to
compose across projects, with minimal boilerplate and strong operational
clarity.

For an implementation example, see examples/service/internal/cli/bind.go.
*/
package healthcheck

import (
	"context"
)

// HealthChecker defines a single health probe operation.
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

// HealthCheckFunc adapts a plain function to the [HealthChecker] interface,
// mirroring the http.HandlerFunc pattern. It removes the need for a wrapper type
// and makes function-based probes (such as one calling [CheckHTTPStatus]) easy to
// register:
//
//	healthcheck.New("upstream", healthcheck.HealthCheckFunc(func(ctx context.Context) error {
//		return healthcheck.CheckHTTPStatus(ctx, client, http.MethodGet, url, http.StatusOK, 2*time.Second)
//	}))
type HealthCheckFunc func(ctx context.Context) error

// HealthCheck calls f(ctx).
func (f HealthCheckFunc) HealthCheck(ctx context.Context) error {
	return f(ctx)
}

// HealthCheck describes one registered probe and its unique identifier.
type HealthCheck struct {
	// ID is a unique identifier for the healthcheck.
	ID string

	// Checker is the function used to perform the healthchecks.
	Checker HealthChecker
}

// New creates a HealthCheck registration entry.
//
// It binds a stable check ID to a HealthChecker implementation so handlers can
// execute and report results consistently. The ID should be unique and
// non-empty within a handler, as results are aggregated by ID.
func New(id string, checker HealthChecker) HealthCheck {
	return HealthCheck{
		ID:      id,
		Checker: checker,
	}
}
