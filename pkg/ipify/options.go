package ipify

import (
	"time"
)

// Option is the interface that allows to set client options.
type Option func(c *Client)

// WithTimeout sets the request timeout used by GetPublicIP.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithURL sets the ipify service endpoint URL.
//
// For IPv6-aware responses, use https://api64.ipify.org.
func WithURL(addr string) Option {
	return func(c *Client) {
		c.apiURL = addr
	}
}

// WithErrorIP sets the fallback IP string returned on failures.
func WithErrorIP(s string) Option {
	return func(c *Client) {
		c.errorIP = s
	}
}

// WithHTTPClient injects a custom HTTP client implementation.
//
// This is useful for testing, tracing, proxies, or custom transports.
func WithHTTPClient(hc HTTPClient) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}
