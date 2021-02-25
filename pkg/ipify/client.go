package ipify

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/nexmoinc/gosrvlib/pkg/logging"
)

const (
	defaultTimeout = 4 * time.Second         // default timeout in seconds
	defaultAPIURL  = "https://api.ipify.org" // use "https://api64.ipify.org" for IPv6 support
	defaultErrorIP = ""                      // string to return in case of error in place of the IP
)

// HTTPClient contains the function to perform the actual HTTP request.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Client represents the config options required by this client.
type Client struct {
	httpClient HTTPClient
	timeout    time.Duration
	apiURL     string
	errorIP    string
}

func defaultClient() *Client {
	return &Client{
		timeout: defaultTimeout,
		apiURL:  defaultAPIURL,
		errorIP: defaultErrorIP,
	}
}

// New creates a new ipify client instance.
func New(opts ...Option) (*Client, error) {
	c := defaultClient()

	for _, applyOpt := range opts {
		applyOpt(c)
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: c.timeout}
	}

	if _, err := url.Parse(c.apiURL); err != nil {
		return nil, fmt.Errorf("invalid service address: %s", c.apiURL)
	}

	return c, nil
}

// GetPublicIP retrieves the public IP of this service instance via ipify.com API.
func (c *Client) GetPublicIP(ctx context.Context) (string, error) {
	ctx, cancelTimeout := context.WithTimeout(ctx, c.timeout)
	defer cancelTimeout()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL, nil)
	if err != nil {
		return c.errorIP, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.httpClient.Do(req) // nolint:bodyclose
	if err != nil {
		return c.errorIP, fmt.Errorf("failed performing ipify request: %w", err)
	}

	defer logging.Close(ctx, resp.Body, "error while closing GetPublicIP response body")

	if resp.StatusCode != http.StatusOK {
		return c.errorIP, fmt.Errorf("unexpected ipify status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.errorIP, fmt.Errorf("failed reading response body: %w", err)
	}

	return string(body), nil
}
