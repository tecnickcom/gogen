package passwordpwned

import (
	"time"
)

// Option is the interface that allows to set client options.
type Option func(c *Client)

// WithURL overrides the default HIBP API base URL (default: https://api.pwnedpasswords.com).
// Useful for routing requests through a self-hosted mirror or a test server.
func WithURL(addr string) Option {
	return func(c *Client) {
		c.apiURL = addr
	}
}

// WithUserAgent overrides the default user-agent for service requests.
func WithUserAgent(s string) Option {
	return func(c *Client) {
		c.userAgent = s
	}
}

// WithTimeout overrides the default request timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithHTTPClient overrides the default HTTP client.
func WithHTTPClient(hc HTTPClient) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithRetryAttempts sets the maximum number of HTTP retry attempts for transient errors.
func WithRetryAttempts(attempts uint) Option {
	return func(c *Client) {
		c.retryAttempts = attempts
	}
}

// WithRetryDelay sets the delay to apply after the first failed attempt.
func WithRetryDelay(value time.Duration) Option {
	return func(c *Client) {
		c.retryDelay = value
	}
}
