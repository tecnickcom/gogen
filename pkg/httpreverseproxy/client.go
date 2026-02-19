package httpreverseproxy

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	libhttputil "github.com/tecnickcom/gogen/pkg/httputil"
	"github.com/tecnickcom/gogen/pkg/logutil"
	"github.com/tecnickcom/gogen/pkg/traceid"
)

// HTTPClient contains the function to perform the HTTP request to the proxied service.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client implements the Reverse Proxy.
type Client struct {
	proxy      *httputil.ReverseProxy
	httpClient HTTPClient
	logger     *slog.Logger
}

// errHandler defines the function signature for error handlers.
type errHandler = func(w http.ResponseWriter, r *http.Request, err error)

// New returns a new instance of the Client.
func New(addr string, opts ...Option) (*Client, error) {
	c := &Client{}

	for _, applyOpt := range opts {
		applyOpt(c)
	}

	if c.proxy == nil {
		c.proxy = &httputil.ReverseProxy{}
	}

	if c.proxy.Rewrite == nil {
		addr = strings.TrimRight(addr, "/")

		proxyURL, err := url.Parse(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid service address: %s", addr)
		}

		c.proxy.Rewrite = func(r *httputil.ProxyRequest) {
			r.SetURL(proxyURL)
			r.Out.URL.Scheme = proxyURL.Scheme
			r.Out.URL.Host = proxyURL.Host
			r.Out.URL.Path = "/" + libhttputil.PathParam(r.Out, "path")
			r.Out.Host = proxyURL.Host
			r.SetXForwarded()
		}
	}

	if c.proxy.Transport == nil {
		if c.httpClient == nil {
			c.httpClient = &http.Client{}
		}

		c.proxy.Transport = &httpWrapper{client: c.httpClient}
	}

	if c.logger == nil {
		c.logger = slog.Default()
	}

	c.proxy.ErrorLog = logutil.NewLogFromSlog(c.logger)

	if c.proxy.ErrorHandler == nil {
		c.proxy.ErrorHandler = defaultErrorHandler(c.logger)
	}

	return c, nil
}

// ForwardRequest forwards a request to the proxied service.
func (c *Client) ForwardRequest(w http.ResponseWriter, r *http.Request) {
	c.proxy.ServeHTTP(w, r) //nolint:gosec
}

// httpWrapper wraps an HTTPClient to implement the RoundTripper interface.
type httpWrapper struct {
	client HTTPClient
}

// RoundTrip implements the RoundTripper interface.
func (c *httpWrapper) RoundTrip(r *http.Request) (*http.Response, error) {
	// Request.RequestURI can't be set in client requests.
	// Ref.: https://github.com/golang/go/blob/f3c39a83a3076eb560c7f687cbb35eef9b506e7d/src/net/http/client.go#L219
	r.RequestURI = ""

	return c.client.Do(r) //nolint:wrapcheck
}

// defaultErrorHandler returns the default error handler for the reverse proxy.
func defaultErrorHandler(logger *slog.Logger) errHandler {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		resTime := time.Now().UTC()
		ctx := r.Context()

		reqTime, ok := libhttputil.GetRequestTimeFromContext(ctx)
		if !ok {
			reqTime = resTime
		}

		logger.With(
			slog.String(traceid.DefaultLogKey, traceid.FromContext(ctx, "")),
			slog.String("request_method", r.Method),
			slog.String("request_path", r.URL.Path),
			slog.String("request_query", r.URL.RawQuery),
			slog.String("request_uri", r.RequestURI),
			slog.Int("response_code", http.StatusBadGateway),
			slog.String("response_message", http.StatusText(http.StatusBadGateway)),
			slog.Any("response_status", libhttputil.Status(http.StatusBadGateway)),
			slog.Time("request_time", reqTime),
			slog.Time("response_time", resTime),
			slog.Duration("response_duration", resTime.Sub(reqTime)),
			slog.Any("error", err),
		).Error("proxy_error")

		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
	}
}
