package passwordpwned

import (
	"context"
	"crypto/sha1" //nolint:gosec
	"fmt"
	"hash"
	"net/http"
	"net/url"
	"time"

	"github.com/tecnickcom/gogen/pkg/httpretrier"
)

const (
	defaultTimeout   = 30 * time.Second
	defaultAPIURL    = "https://api.pwnedpasswords.com"
	rangePath        = "range"
	defaultUserAgent = "gogen.passwordpwned/1"
)

// HTTPClient is the minimal HTTP transport used by [Client].
type HTTPClient interface {
	// Do sends an HTTP request and returns an HTTP response.
	Do(req *http.Request) (*http.Response, error)
}

// Client queries the HIBP Pwned Passwords API using the k-Anonymity range endpoint.
// Create one with [New]; the zero value is not usable.
type Client struct {
	httpClient    HTTPClient
	timeout       time.Duration
	retryDelay    time.Duration
	retryAttempts uint
	hashObj       hash.Hash
	apiURL        string
	userAgent     string
}

// defaultClient returns a client preconfigured with package defaults.
func defaultClient() *Client {
	return &Client{
		timeout:       defaultTimeout,
		retryAttempts: httpretrier.DefaultAttempts,
		retryDelay:    httpretrier.DefaultDelay,
		hashObj:       sha1.New(), //nolint:gosec
		apiURL:        defaultAPIURL,
		userAgent:     defaultUserAgent,
	}
}

// New creates a [Client] with the given options applied over sensible defaults.
//
// Defaults:
//   - API URL:        https://api.pwnedpasswords.com
//   - Timeout:        30 s
//   - User-Agent:     gogen.passwordpwned/1
//   - Retry attempts: httpretrier.DefaultAttempts
//
// Use [WithURL], [WithTimeout], [WithHTTPClient], [WithUserAgent],
// [WithRetryAttempts], or [WithRetryDelay] to override individual settings.
func New(opts ...Option) (*Client, error) {
	c := defaultClient()

	for _, applyOpt := range opts {
		applyOpt(c)
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: c.timeout}
	}

	_, err := url.Parse(c.apiURL)
	if err != nil {
		return nil, fmt.Errorf("invalid service address: %s", c.apiURL)
	}

	return c, nil
}

// HealthCheck reports backend readiness for this client.
// The HIBP range API has no dedicated ping endpoint, so this method currently returns nil.
func (c *Client) HealthCheck(_ context.Context) error {
	return nil
}

// newHTTPRetrier builds a read-oriented retrier for range endpoint requests.
func (c *Client) newHTTPRetrier() (*httpretrier.HTTPRetrier, error) {
	//nolint:wrapcheck
	return httpretrier.New(
		c.httpClient,
		httpretrier.WithRetryIfFn(httpretrier.RetryIfForReadRequests),
		httpretrier.WithAttempts(c.retryAttempts),
	)
}
