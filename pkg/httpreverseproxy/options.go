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
