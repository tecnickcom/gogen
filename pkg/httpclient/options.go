package httpclient

import (
	"net/http"
	"time"
)

// InstrumentRoundTripper is an alias for a RoundTripper function.
type InstrumentRoundTripper func(next http.RoundTripper) http.RoundTripper

// RedactFn is an alias for a redact function.
type RedactFn func(s string) string

// Option is the interface that allows to set client options.
type Option func(c *Client)

// WithTimeout overrides the default client timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.client.Timeout = timeout
	}
}

// WithRoundTripper wraps the HTTP client Transport with the specified RoundTripper function.
func WithRoundTripper(fn InstrumentRoundTripper) Option {
	return func(c *Client) {
		c.client.Transport = fn(http.DefaultTransport)
	}
}

// WithTraceIDHeaderName sets the trace id header name.
func WithTraceIDHeaderName(name string) Option {
	return func(c *Client) {
		c.traceIDHeaderName = name
	}
}

// WithComponent sets the component name to be used in logs.
func WithComponent(name string) Option {
	return func(c *Client) {
		c.component = name
	}
}

// WithReadactFn set the function used to redact HTTP request and response dumps in the logs.
func WithReadactFn(fn RedactFn) Option {
	return func(c *Client) {
		c.redactFn = fn
	}
}
