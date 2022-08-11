package slack

import (
	"time"
)

// Option is the interface that allows to set client options.
type Option func(c *Client)

// WithTimeout overrides the default request timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithPingTimeout overrides the default ping timeout.
func WithPingTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.pingTimeout = timeout
	}
}

// WithPingURL overrides the default ping timeout.
func WithPingURL(pingURL string) Option {
	return func(c *Client) {
		c.pingURL = pingURL
	}
}

// WithHTTPClient overrides the default HTTP client.
func WithHTTPClient(hc HTTPClient) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithRetryAttempts overrides the default HTTP client.
func WithRetryAttempts(attempts uint) Option {
	return func(c *Client) {
		c.retryAttempts = attempts
	}
}
