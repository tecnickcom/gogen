package opentel

import (
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Option is the interface that allows to set client options.
type Option func(c *Client) error

// WithTracerProvider overrides the default TracerProvider.
func WithTracerProvider(p *sdktrace.TracerProvider) Option {
	return func(c *Client) error {
		c.tracerProvider = p
		return nil
	}
}

// WithMeterProvider overrides the default MeterProvider.
func WithMeterProvider(p *sdkmetric.MeterProvider) Option {
	return func(c *Client) error {
		c.meterProvider = p
		return nil
	}
}
