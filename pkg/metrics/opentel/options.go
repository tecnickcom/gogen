package opentel

import (
	"go.opentelemetry.io/otel/propagation"
)

// Option is the interface that allows to set client options.
type Option func(c *Client) error

// WithSDKResourceFn overrides the default SDKResourceFunc.
func WithSDKResourceFn(fn SDKResourceFunc) Option {
	return func(c *Client) error {
		c.resFn = fn
		return nil
	}
}

// WithTracerProviderFn overrides the default TraceProviderFunc.
func WithTracerProviderFn(fn TraceProviderFunc) Option {
	return func(c *Client) error {
		c.tracerProviderFn = fn
		return nil
	}
}

// WithMeterProviderFn overrides the default MetricProviderFunc.
func WithMeterProviderFn(fn MetricProviderFunc) Option {
	return func(c *Client) error {
		c.meterProviderFn = fn
		return nil
	}
}

// WithPropagator overrides the default TextMapPropagator.
func WithPropagator(p propagation.TextMapPropagator) Option {
	return func(c *Client) error {
		c.propagator = p
		return nil
	}
}
