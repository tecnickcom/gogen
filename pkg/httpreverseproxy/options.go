package httpreverseproxy

import (
	"log/slog"
	"net/http/httputil"
	"time"
)

// RedactFn is an alias for a redact function that takes raw bytes and returns a redacted string.
type RedactFn func(b []byte) string

// Option is the interface that allows to set client options.
type Option func(c *Client)

// WithReverseProxy replaces the default httputil.ReverseProxy.
//
// New auto-configures only the fields left unset: it installs the default rewrite
// when both Rewrite and Director are nil, a default upstream transport when
// Transport is nil, and a default ErrorLog/ErrorHandler when those are nil. Provide
// a Rewrite or a Director (never both, which ReverseProxy rejects) to fully control
// request rewriting; leave them nil for the default upstream routing. Advanced
// fields New never touches — FlushInterval, ModifyResponse, and a non-nil
// ErrorHandler — are honored as configured, so pass a fully-built proxy to customize
// response streaming or mutation.
func WithReverseProxy(p *httputil.ReverseProxy) Option {
	return func(c *Client) {
		c.proxy = p
	}
}

// WithHTTPClient customizes HTTP client for upstream forwarding requests.
func WithHTTPClient(h HTTPClient) Option {
	return func(c *Client) {
		c.httpClient = h
	}
}

// WithLogger overrides default logger for proxy event logging.
func WithLogger(l *slog.Logger) Option {
	return func(c *Client) {
		c.logger = l
	}
}

// WithPathParam sets the name of the catch-all route parameter that holds the
// upstream path used by the default rewrite (e.g. "path" for a route registered
// as "/proxy/*path"). An empty value falls back to the default ("path").
//
// It has no effect when a custom rewrite is provided via WithReverseProxy.
func WithPathParam(name string) Option {
	return func(c *Client) {
		if name == "" {
			name = defaultPathParam
		}

		c.pathParam = name
	}
}

// WithRedactFn customizes redaction of sensitive data written to logs, namely the
// request query string and the upstream URL embedded in error entries.
//
// A nil fn is ignored, keeping the default redaction, so the option can never
// install a nil function that would panic while logging. fn may be called
// concurrently by simultaneous forwarded requests, so it must be safe for
// concurrent use. The default redact.HTTPDataString is.
func WithRedactFn(fn RedactFn) Option {
	return func(c *Client) {
		if fn == nil {
			return
		}

		c.redactFn = fn
	}
}

// WithLaxBasePath disables the default base-path boundary check, forwarding "." and
// ".." path segments verbatim.
//
// By default, when the upstream address carries a base path, a proxied request whose
// path resolves outside that base path is rejected with HTTP 400 before the upstream
// is contacted, so the base path acts as a boundary. This option restores transparent
// forwarding: use it only for a pass-through proxy where the upstream itself is the
// authorization boundary, since a client can then traverse to sibling upstream paths.
//
// It only affects the default rewrite with a non-empty base path; it is a no-op with
// a custom rewrite/director or when the address carries no base path.
func WithLaxBasePath() Option {
	return func(c *Client) {
		c.laxBasePath = true
	}
}

// WithFlushInterval sets httputil.ReverseProxy.FlushInterval, the flush cadence when
// copying the upstream response body to the client.
//
// A negative value flushes immediately after each write (useful for low-latency
// streaming of content types ReverseProxy does not already flush eagerly; it always
// flushes text/event-stream). The zero default leaves ReverseProxy's behavior
// unchanged and does not override a value set via WithReverseProxy.
func WithFlushInterval(d time.Duration) Option {
	return func(c *Client) {
		c.flushInterval = d
	}
}
