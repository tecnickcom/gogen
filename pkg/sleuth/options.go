package sleuth

import (
	"time"
)

// Option customizes Sleuth client configuration.
type Option func(c *Client)

// WithTimeout overrides default request timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithPingTimeout overrides default health-check timeout.
func WithPingTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.pingTimeout = timeout
	}
}

// WithHTTPClient injects a custom HTTP client implementation.
func WithHTTPClient(hc HTTPClient) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithRetryAttempts overrides retry attempt count for write operations.
func WithRetryAttempts(attempts uint) Option {
	return func(c *Client) {
		c.retryAttempts = attempts
	}
}

// WithRetryDelay sets base delay for retrier backoff configuration.
func WithRetryDelay(value time.Duration) Option {
	return func(c *Client) {
		c.retryDelay = value
	}
}
