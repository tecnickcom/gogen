package ipify

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

// WithURL overrides the default service URL.
func WithURL(addr string) Option {
	return func(c *Client) {
		c.apiURL = addr
	}
}

// WithErrorIP overrides the default error return string.
func WithErrorIP(s string) Option {
	return func(c *Client) {
		c.errorIP = s
	}
}

// WithHTTPClient overrides the default HTTP client.
func WithHTTPClient(hc HTTPClient) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}
