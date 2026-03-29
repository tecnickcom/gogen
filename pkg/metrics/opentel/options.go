package opentel

import (
	"go.opentelemetry.io/otel/propagation"
)

// Option configures a [Client] during [New].
type Option func(c *Client) error

// WithSDKResourceFn overrides resource construction (service attributes,
// environment metadata, etc.).
func WithSDKResourceFn(fn SDKResourceFunc) Option {
	return func(c *Client) error {
		c.resFn = fn
		return nil
	}
}

// WithTracerProviderFn overrides tracer provider construction (exporter,
// batching strategy, resource binding).
func WithTracerProviderFn(fn TraceProviderFunc) Option {
	return func(c *Client) error {
		c.tracerProviderFn = fn
		return nil
	}
}

// WithMeterProviderFn overrides meter provider construction (exporter,
// reader interval, resource binding).
func WithMeterProviderFn(fn MetricProviderFunc) Option {
	return func(c *Client) error {
		c.meterProviderFn = fn
		return nil
	}
}

// WithPropagator overrides the default context propagator used for cross-
// service trace propagation.
func WithPropagator(p propagation.TextMapPropagator) Option {
	return func(c *Client) error {
		c.propagator = p
		return nil
	}
}
