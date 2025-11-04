package passwordpwned

import (
	"context"
	"crypto/sha1" //nolint:gosec
	"fmt"
	"hash"
	"net/http"
	"net/url"
	"time"

	"github.com/Vonage/gosrvlib/pkg/httpretrier"
)

const (
	defaultTimeout   = 30 * time.Second
	defaultAPIURL    = "https://api.pwnedpasswords.com"
	rangePath        = "range"
	defaultUserAgent = "gosrvlib.passwordpwned/1"
)

// HTTPClient contains the function to perform the actual HTTP request.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client represents the config options required by the Accounts client.
type Client struct {
	httpClient    HTTPClient
	timeout       time.Duration
	retryDelay    time.Duration
	retryAttempts uint
	hashObj       hash.Hash
	apiURL        string
	userAgent     string
}

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

// New creates a new client instance.
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

// HealthCheck performs a status check on this service.
func (c *Client) HealthCheck(_ context.Context) error {
	return nil
}

func (c *Client) newHTTPRetrier() (*httpretrier.HTTPRetrier, error) {
	//nolint:wrapcheck
	return httpretrier.New(
		c.httpClient,
		httpretrier.WithRetryIfFn(httpretrier.RetryIfForReadRequests),
		httpretrier.WithAttempts(c.retryAttempts),
	)
}
