package slack

import (
	"time"
)

// Option applies a configuration change to [Client].
type Option func(c *Client)

// WithTimeout sets webhook request timeout.
//
// It is applied only to the default HTTP client that New creates; it has no
// effect when a custom client is supplied via WithHTTPClient (that client owns
// its own timeout).
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithPingTimeout sets timeout used by HealthCheck.
func WithPingTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.pingTimeout = timeout
	}
}

// WithPingURL overrides the Slack status endpoint used by HealthCheck.
func WithPingURL(pingURL string) Option {
	return func(c *Client) {
		c.pingURL = pingURL
	}
}

// WithHTTPClient injects a custom HTTP client implementation.
func WithHTTPClient(hc HTTPClient) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithRetryAttempts sets max retry attempts for webhook sends.
func WithRetryAttempts(attempts uint) Option {
	return func(c *Client) {
		c.retryAttempts = attempts
	}
}

// WithRetryDelay sets the base delay between a failed webhook send and its
// retry (default: [httpretrier.DefaultDelay], 1 s). Must be positive ([New]
// rejects non-positive values).
func WithRetryDelay(value time.Duration) Option {
	return func(c *Client) {
		c.retryDelay = value
	}
}
