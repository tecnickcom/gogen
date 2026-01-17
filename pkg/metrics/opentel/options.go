package opentel

import (
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

// Option is the interface that allows to set client options.
type Option func(c *Client) error

// WithTracerProvider overrides the default TracerProvider.
func WithTracerProvider(p *trace.TracerProvider) Option {
	return func(c *Client) error {
		c.tracerProvider = p
		return nil
	}
}

// WithMeterProvider overrides the default MeterProvider.
func WithMeterProvider(p *metric.MeterProvider) Option {
	return func(c *Client) error {
		c.meterProvider = p
		return nil
	}
}
