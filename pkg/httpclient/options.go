package httpclient

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// InstrumentRoundTripper is an alias for a RoundTripper function.
type InstrumentRoundTripper func(next http.RoundTripper) http.RoundTripper

// DialContextFunc is an alias for a net.Dialer.DialContext function.
type DialContextFunc func(ctx context.Context, network, address string) (net.Conn, error)

// RedactFn is an alias for a redact function that takes raw bytes and returns a redacted string.
type RedactFn func(b []byte) string

// Option is the interface that allows to set client options.
type Option func(c *Client)

// WithTimeout customizes request timeout (default 1 minute).
//
// A zero or negative timeout disables the timeout (the net/http convention),
// leaving the request bounded only by its own context.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithRoundTripper wraps client transport with custom RoundTripper for middleware instrumentation.
//
// A nil fn, or an fn that returns nil, is ignored and leaves the current
// transport in place (honoring the no-panics policy rather than deferring a
// nil-transport failure to the first request).
//
// Ordering matters relative to WithDialContext: WithRoundTripper replaces the
// transport with the (typically non-*http.Transport) wrapper returned by fn,
// after which WithDialContext can no longer find the underlying *http.Transport
// and becomes a silent no-op. Apply WithDialContext before WithRoundTripper.
func WithRoundTripper(fn InstrumentRoundTripper) Option {
	return func(c *Client) {
		if fn == nil {
			return
		}

		if wrapped := fn(c.client.Transport); wrapped != nil {
			c.client.Transport = wrapped
		}
	}
}

// WithTraceIDHeaderName specifies custom trace ID header name (default X-Request-ID).
//
// An empty name is ignored (the default is kept) since an empty header name
// would make every request fail at send time.
func WithTraceIDHeaderName(name string) Option {
	return func(c *Client) {
		if name == "" {
			return
		}

		c.traceIDHeaderName = name
	}
}

// WithComponent customizes component name embedded in log field entries.
//
// An empty name is ignored so the default component tag is preserved.
func WithComponent(name string) Option {
	return func(c *Client) {
		if name == "" {
			return
		}

		c.component = name
	}
}

// WithRedactFn customizes sensitive data redaction applied to debug-level payload dumps and the logged query string.
//
// A nil fn is ignored, keeping the default redaction, so the option can never
// install a nil function that would panic on the first debug-level request.
//
// fn is called concurrently by simultaneous Do calls (at least once per request
// for the query string, plus once per dump at debug level), so it must be safe
// for concurrent use. The default redact.HTTPDataString is.
func WithRedactFn(fn RedactFn) Option {
	return func(c *Client) {
		if fn == nil {
			return
		}

		c.redactFn = fn
	}
}

// WithLogPrefix specifies a prefix applied to all log field names in Do (e.g., "http_").
//
// Only field names are prefixed; the log message itself is a stable constant so
// entries remain filterable regardless of the prefix. An empty prefix (the
// default) leaves field names unprefixed.
func WithLogPrefix(prefix string) Option {
	return func(c *Client) {
		c.logPrefix = prefix
	}
}

// WithMaxDumpSize caps the request/response body size (in bytes, measured by the
// advertised Content-Length) buffered into debug-level payload dumps.
//
// Bodies larger than the cap, or of unknown length (streaming/chunked requests),
// have their headers dumped but their payload omitted. This bounds memory use
// and avoids a deadlock when dumping a streaming request body. A non-positive
// value disables the cap. The default is 1 MiB.
func WithMaxDumpSize(n int64) Option {
	return func(c *Client) {
		c.maxDumpSize = n
	}
}

// WithTransport replaces the client's base transport (a private clone of
// http.DefaultTransport) with a clone of t, so callers can tune connection
// pooling (MaxIdleConnsPerHost, MaxConnsPerHost, timeouts), TLS, proxy, and
// HTTP/2 settings. The transport is cloned, so later mutations of t do not affect
// the client and vice versa. A nil transport is ignored.
//
// Apply this before WithDialContext and WithRoundTripper: WithDialContext mutates
// the base *http.Transport installed here, and WithRoundTripper wraps it. Applied
// after WithDialContext it would discard the custom dialer; applied after
// WithRoundTripper it would discard the wrapper.
func WithTransport(t *http.Transport) Option {
	return func(c *Client) {
		if t == nil {
			return
		}

		c.client.Transport = t.Clone()
	}
}

// WithDialContext customizes network connection establishment via transport DialContext hook.
//
// A nil fn is ignored. It mutates the client's own *http.Transport (a private
// clone of http.DefaultTransport), so it never affects http.DefaultTransport or
// any other client.
//
// Ordering matters relative to WithRoundTripper: this option only takes effect
// while the client's transport is still an *http.Transport. Once
// WithRoundTripper has wrapped the transport, this option can no longer reach
// the underlying *http.Transport and silently does nothing. Apply
// WithDialContext before WithRoundTripper.
func WithDialContext(fn DialContextFunc) Option {
	return func(c *Client) {
		if fn == nil {
			return
		}

		t, ok := c.client.Transport.(*http.Transport)
		if ok {
			t.DialContext = fn
		}
	}
}

// WithTLSClientConfig sets the TLS configuration on the client's base transport,
// for custom CAs, client certificates (mTLS), or other TLS tuning, without having
// to rebuild a transport from scratch (which would drop the default dial, idle,
// HTTP/2, and proxy settings).
//
// A nil config is ignored, keeping the transport's default TLS behavior. Like
// WithDialContext it mutates the client's own *http.Transport, so it takes effect
// only while the transport is still an *http.Transport: apply it after
// WithTransport (which installs the base) and before WithRoundTripper (which
// wraps the transport, after which this option silently does nothing). The config
// is stored by reference; do not mutate it after the client is created.
func WithTLSClientConfig(cfg *tls.Config) Option {
	return func(c *Client) {
		if cfg == nil {
			return
		}

		t, ok := c.client.Transport.(*http.Transport)
		if ok {
			t.TLSClientConfig = cfg
		}
	}
}

// WithLogger overrides default logger for all request/response logging.
//
// A nil logger is ignored so the default logger is never replaced with one that
// would panic on the first request.
func WithLogger(logger *slog.Logger) Option {
	return func(c *Client) {
		if logger == nil {
			return
		}

		c.logger = logger
	}
}
