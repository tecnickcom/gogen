package httpclient

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// InstrumentRoundTripper is an alias for a RoundTripper function.
type InstrumentRoundTripper func(next http.RoundTripper) http.RoundTripper

// DialContextFunc is an alias for a net.Dialer.DialContext function.
type DialContextFunc func(ctx context.Context, network, address string) (net.Conn, error)

// RedactFn is an alias for a redact function.
type RedactFn func(s string) string

// Option is the interface that allows to set client options.
type Option func(c *Client)

// WithTimeout customizes request timeout (default 1 minute).
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.client.Timeout = timeout
	}
}

// WithRoundTripper wraps client transport with custom RoundTripper for middleware instrumentation.
func WithRoundTripper(fn InstrumentRoundTripper) Option {
	return func(c *Client) {
		c.client.Transport = fn(c.client.Transport)
	}
}

// WithTraceIDHeaderName specifies custom trace ID header name (default X-Request-ID).
func WithTraceIDHeaderName(name string) Option {
	return func(c *Client) {
		c.traceIDHeaderName = name
	}
}

// WithComponent customizes component name embedded in log field entries.
func WithComponent(name string) Option {
	return func(c *Client) {
		c.component = name
	}
}

// WithRedactFn customizes sensitive data redaction applied to debug-level payload dumps.
func WithRedactFn(fn RedactFn) Option {
	return func(c *Client) {
		c.redactFn = fn
	}
}

// WithLogPrefix specifies prefix for all log field names in Do (e.g., "http_").
func WithLogPrefix(prefix string) Option {
	return func(c *Client) {
		c.logPrefix = prefix
	}
}

// WithDialContext customizes network connection establishment via transport DialContext hook.
func WithDialContext(fn DialContextFunc) Option {
	return func(c *Client) {
		t, ok := c.client.Transport.(*http.Transport)
		if ok {
			t.DialContext = fn
		}
	}
}

// WithLogger overrides default logger for all request/response logging.
func WithLogger(logger *slog.Logger) Option {
	return func(c *Client) {
		c.logger = logger
	}
}
