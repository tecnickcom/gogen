package jirasrv

import (
	"time"
)

// Option is the interface that allows to set client options.
type Option func(c *Client)

// WithTimeout sets the default timeout for Jira API requests.
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

// WithHTTPClient injects a custom HTTP client implementation.
//
// Use this for custom transport behavior or test doubles.
func WithHTTPClient(hc HTTPClient) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithRetryAttempts sets the maximum retry attempts for API requests.
func WithRetryAttempts(attempts uint) Option {
	return func(c *Client) {
		c.retryAttempts = attempts
	}
}

// WithRetryDelay sets the delay between retry attempts.
func WithRetryDelay(value time.Duration) Option {
	return func(c *Client) {
		c.retryDelay = value
	}
}
