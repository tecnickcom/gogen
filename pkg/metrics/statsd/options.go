package statsd

import (
	"time"
)

// Option is the interface that allows to set client options.
type Option func(c *Client)

// WithPrefix sets the StatsD client's string prefix that will be used in every bucket name.
func WithPrefix(prefix string) Option {
	return func(c *Client) {
		c.prefix = prefix
	}
}

// WithNetwork sets the network type used by the StatsD client (i.e. udp or tcp).
func WithNetwork(network string) Option {
	return func(c *Client) {
		c.network = network
	}
}

// WithAddress sets the network address of the StatsD daemon (ip:port) or just (:port).
func WithAddress(address string) Option {
	return func(c *Client) {
		c.address = address
	}
}

// WithFlushPeriod sets how often the StatsD client's buffer is flushed.
func WithFlushPeriod(flushPeriod time.Duration) Option {
	return func(c *Client) {
		c.flushPeriod = flushPeriod
	}
}
