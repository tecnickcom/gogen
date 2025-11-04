package httpclient

import (
	"context"
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

// WithTimeout overrides the default client timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.client.Timeout = timeout
	}
}

// WithRoundTripper wraps the HTTP client Transport with the specified RoundTripper function.
func WithRoundTripper(fn InstrumentRoundTripper) Option {
	return func(c *Client) {
		c.client.Transport = fn(c.client.Transport)
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

// WithRedactFn set the function used to redact HTTP request and response dumps in the logs.
func WithRedactFn(fn RedactFn) Option {
	return func(c *Client) {
		c.redactFn = fn
	}
}

// WithLogPrefix specifies a string prefix to be added to each log field name in the Do method.
func WithLogPrefix(prefix string) Option {
	return func(c *Client) {
		c.logPrefix = prefix
	}
}

// WithDialContext sets the DialContext function for the HTTP client.
// The DialContext function is used to establish network connections.
// It allows customizing the behavior of the client's underlying transport.
func WithDialContext(fn DialContextFunc) Option {
	return func(c *Client) {
		t, ok := c.client.Transport.(*http.Transport)
		if ok {
			t.DialContext = fn
		}
	}
}
