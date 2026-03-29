package statsd

import (
	"time"
)

// Option configures a [Client].
type Option func(c *Client)

// WithPrefix sets the StatsD client's string prefix that will be used in every bucket name.
func WithPrefix(prefix string) Option {
	return func(c *Client) {
		c.prefix = prefix
	}
}

// WithNetwork sets the StatsD transport network (typically "udp" or "tcp").
func WithNetwork(network string) Option {
	return func(c *Client) {
		c.network = network
	}
}

// WithAddress sets the StatsD daemon address (host:port or :port).
func WithAddress(address string) Option {
	return func(c *Client) {
		c.address = address
	}
}

// WithFlushPeriod sets how often buffered metrics are flushed to the daemon.
// When set to 0, flush occurs only when the buffer is full.
func WithFlushPeriod(flushPeriod time.Duration) Option {
	return func(c *Client) {
		c.flushPeriod = flushPeriod
	}
}
