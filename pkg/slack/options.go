package slack

import (
	"time"
)

// Option applies a configuration change to [Client].
type Option func(c *Client)

// WithTimeout sets webhook request timeout.
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
