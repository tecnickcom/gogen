package metrics

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
		return c.Registry.Register(m)
	}
}
