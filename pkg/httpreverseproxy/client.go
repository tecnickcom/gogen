package httpreverseproxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	libhttputil "github.com/nexmoinc/gosrvlib/pkg/httputil"
)

// HTTPClient contains the function to perform the HTTP request to the proxied service.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Client implements the Reverse Proxy.
type Client struct {
	proxy      *httputil.ReverseProxy
	httpClient HTTPClient
}

// New returns a new instance of the Client.
func New(addr string, opts ...Option) (*Client, error) {
	c := &Client{}

	for _, applyOpt := range opts {
		applyOpt(c)
	}

	if c.proxy == nil {
		c.proxy = &httputil.ReverseProxy{}
	}

	if c.proxy.Director == nil {
		addr = strings.TrimRight(addr, "/")

		proxyURL, err := url.Parse(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid service address: %s", addr)
		}

		c.proxy.Director = func(r *http.Request) {
			r.URL.Scheme = proxyURL.Scheme
			r.URL.Host = proxyURL.Host
			r.URL.Path = "/" + libhttputil.PathParam(r, "path")
			r.Host = proxyURL.Host
			r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
		}
	}

	if c.proxy.Transport == nil {
		if c.httpClient == nil {
			c.httpClient = &http.Client{}
		}

		c.proxy.Transport = &httpWrapper{client: c.httpClient}
	}

	return c, nil
}

// ForwardRequest forwards a request to the proxied service.
func (c *Client) ForwardRequest(w http.ResponseWriter, r *http.Request) {
	c.proxy.ServeHTTP(w, r)
}

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
