package devlake

import (
	"time"
)

// Option is the interface that allows to set client options.
type Option func(c *Client)

// WithTimeout sets the default HTTP timeout for regular API requests.
//
// It is applied only to the default HTTP client that New creates; it has no
// effect when a custom client is supplied via WithHTTPClient (that client owns
// its own timeout).
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithPingTimeout sets the timeout used by HealthCheck.
func WithPingTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.pingTimeout = timeout
	}
}

// WithPingURL overrides the health-check endpoint URL.
//
// This is useful when DevLake is exposed through custom routing paths.
func WithPingURL(pingURL string) Option {
	return func(c *Client) {
		c.pingURL = pingURL
	}
}

// WithHTTPClient injects a custom HTTP client implementation.
//
// Use this for advanced transports, tracing, or test doubles.
func WithHTTPClient(hc HTTPClient) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithRetryAttempts sets the maximum retry attempts for write requests.
func WithRetryAttempts(attempts uint) Option {
	return func(c *Client) {
		c.retryAttempts = attempts
	}
}

// WithRetryDelay sets the delay applied between retry attempts.
func WithRetryDelay(value time.Duration) Option {
	return func(c *Client) {
		c.retryDelay = value
	}
}
