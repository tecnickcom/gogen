package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Option is the interface that allows to set client options.
type Option func(c *Client) error

// WithHandlerOpts sets the options how to serve metrics via an http.Handler.
// The zero value of HandlerOpts is a reasonable default.
func WithHandlerOpts(opts promhttp.HandlerOpts) Option {
	return func(c *Client) error {
		c.handlerOpts = opts
		return nil
	}
}

// WithCollector register a new generic collector.
func WithCollector(m prometheus.Collector) Option {
	return func(c *Client) error {
		return c.registry.Register(m) // nolint:wrapcheck
	}
}

// WithInboundRequestSizeBuckets set the buckets size in bytes for the inbound requests.
func WithInboundRequestSizeBuckets(buckets []float64) Option {
	return func(c *Client) error {
		c.inboundRequestSizeBuckets = buckets
		return nil
	}
}

// WithInboundResponseSizeBuckets set the buckets size in bytes for the inbound response.
func WithInboundResponseSizeBuckets(buckets []float64) Option {
	return func(c *Client) error {
		c.inboundResponseSizeBuckets = buckets
		return nil
	}
}

// WithInboundRequestDurationBuckets set the buckets size in seconds for the inbound requests duration.
func WithInboundRequestDurationBuckets(buckets []float64) Option {
	return func(c *Client) error {
		c.inboundRequestDurationBuckets = buckets
		return nil
	}
}

// WithOutboundRequestDurationBuckets set the buckets size in seconds for the outbound requests duration.
func WithOutboundRequestDurationBuckets(buckets []float64) Option {
	return func(c *Client) error {
		c.outboundRequestDurationBuckets = buckets
		return nil
	}
}
