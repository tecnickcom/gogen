package httpreverseproxy

import (
	"log/slog"
	"net/http/httputil"
)

// Option is the interface that allows to set client options.
type Option func(c *Client)

// WithReverseProxy replaces default httputil.ReverseProxy; leave Director/Transport nil for auto-configuration.
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
