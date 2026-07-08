package passwordpwned

import (
	"time"
)

// Option customizes passwordpwned client configuration.
type Option func(c *Client)

// WithURL overrides the default HIBP API base URL (default: https://api.pwnedpasswords.com).
// Useful for routing requests through a self-hosted mirror or a test server.
// A trailing slash is trimmed; a URL with a query or fragment is rejected by
// [New]. Hash-suffix matching is case-insensitive: responses from lowercase-hex
// mirrors are normalized before matching.
func WithURL(addr string) Option {
	return func(c *Client) {
		c.apiURL = addr
	}
}

// WithUserAgent overrides the User-Agent header used by API requests.
// The value must be non-empty and free of control characters ([New] rejects
// invalid values): Go suppresses an empty User-Agent header entirely, the HIBP
// API rejects requests without one, and the transport rejects control bytes.
func WithUserAgent(s string) Option {
	return func(c *Client) {
		c.userAgent = s
	}
}

// WithTimeout sets the HTTP request timeout of the internally constructed HTTP
// client (default: 30 s). It has no effect when [WithHTTPClient] is used —
// configure the injected client's timeout directly. Non-positive values are
// ignored, keeping the default: net/http would treat them as "no timeout",
// leaving requests bounded only by the caller's context.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		if timeout > 0 {
			c.timeout = timeout
		}
	}
}

// WithHTTPClient injects a custom HTTP client implementation.
// The injected client's own timeout applies ([WithTimeout] is ignored); if it
// has none, requests are bounded only by the caller's context.
func WithHTTPClient(hc HTTPClient) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithRetryAttempts sets the maximum number of total attempts — the initial
// request plus retries (default: [httpretrier.DefaultAttempts], 4). Must be at
// least 1 ([New] rejects 0); a value of 1 disables retries.
func WithRetryAttempts(attempts uint) Option {
	return func(c *Client) {
		c.retryAttempts = attempts
	}
}

// WithRetryDelay sets the base delay between a failed attempt and its retry
// (default: [httpretrier.DefaultDelay], 1 s). Must be positive ([New] rejects
// non-positive values).
func WithRetryDelay(value time.Duration) Option {
	return func(c *Client) {
		c.retryDelay = value
	}
}

// WithResponseSizeLimit caps the decoded response size (in bytes) to guard
// against decompression-bomb style memory exhaustion. Non-positive values are
// ignored, keeping the default limit. Useful when routing through a mirror that
// legitimately returns larger responses.
func WithResponseSizeLimit(limit int64) Option {
	return func(c *Client) {
		if limit > 0 {
			c.maxResponseBytes = limit
		}
	}
}

// WithPingTimeout sets the per-probe timeout of [Client.HealthCheck]
// (default: 5 s). Non-positive values are ignored, keeping the default.
func WithPingTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		if timeout > 0 {
			c.pingTimeout = timeout
		}
	}
}
