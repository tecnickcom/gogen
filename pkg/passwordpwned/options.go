package passwordpwned

import (
	"time"
)

// Option customizes passwordpwned client configuration.
type Option func(c *Client)

// WithURL overrides the default HIBP API base URL (default: https://api.pwnedpasswords.com).
// Useful for routing requests through a self-hosted mirror or a test server.
func WithURL(addr string) Option {
	return func(c *Client) {
		c.apiURL = addr
	}
}

// WithUserAgent overrides the User-Agent header used by API requests.
func WithUserAgent(s string) Option {
	return func(c *Client) {
		c.userAgent = s
	}
}

// WithTimeout sets HTTP request timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithHTTPClient injects a custom HTTP client implementation.
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

// WithRetryDelay sets base delay for retry backoff.
func WithRetryDelay(value time.Duration) Option {
	return func(c *Client) {
		c.retryDelay = value
	}
}
