package ipify

import (
	"time"
)

// ClientOption is the interface that allows to set client options.
type ClientOption func(c *Client)

// WithTimeout overrides the default request timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithURL overrides the default service URL.
func WithURL(addr string) ClientOption {
	return func(c *Client) {
		c.apiURL = addr
	}
}

// WithErrorIP overrides the default error return string.
func WithErrorIP(s string) ClientOption {
	return func(c *Client) {
		c.errorIP = s
	}
}

// WithHTTPClient overrides the default HTTP client.
func WithHTTPClient(hc HTTPClient) ClientOption {
	return func(c *Client) {
		c.httpClient = hc
	}
}
