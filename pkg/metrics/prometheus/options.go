package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Option configures a [Client] during [New].
type Option func(c *Client) error

// WithHandlerOpts sets the options how to serve metrics via an http.Handler.
// The zero value of HandlerOpts is a reasonable default.
func WithHandlerOpts(opts promhttp.HandlerOpts) Option {
	return func(c *Client) error {
		c.handlerOpts = opts
		return nil
	}
}

// WithCollector registers an additional Prometheus collector in the client's
// registry.
func WithCollector(m prometheus.Collector) Option {
	return func(c *Client) error {
		return c.registry.Register(m)
	}
}

// WithInboundRequestSizeBuckets sets histogram buckets (in bytes) for inbound
// HTTP request size metrics.
func WithInboundRequestSizeBuckets(buckets []float64) Option {
	return func(c *Client) error {
		c.inboundRequestSizeBuckets = buckets
		return nil
	}
}

// WithInboundResponseSizeBuckets sets histogram buckets (in bytes) for inbound
// HTTP response size metrics.
func WithInboundResponseSizeBuckets(buckets []float64) Option {
	return func(c *Client) error {
		c.inboundResponseSizeBuckets = buckets
		return nil
	}
}

// WithInboundRequestDurationBuckets sets histogram buckets (in seconds) for
// inbound HTTP request duration metrics.
func WithInboundRequestDurationBuckets(buckets []float64) Option {
	return func(c *Client) error {
		c.inboundRequestDurationBuckets = buckets
		return nil
	}
}

// WithOutboundRequestDurationBuckets sets histogram buckets (in seconds) for
// outbound HTTP request duration metrics.
func WithOutboundRequestDurationBuckets(buckets []float64) Option {
	return func(c *Client) error {
		c.outboundRequestDurationBuckets = buckets
		return nil
	}
}
