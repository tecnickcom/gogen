package ipify

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Default configuration values.
const (
	defaultTimeout = 4 * time.Second         // default timeout in seconds
	defaultAPIURL  = "https://api.ipify.org" // use "https://api64.ipify.org" for IPv6 support
	defaultErrorIP = ""                      // string to return in case of error in place of the IP
)

// HTTPClient contains the function to perform the actual HTTP request.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client represents the config options required by this client.
type Client struct {
	httpClient HTTPClient
	timeout    time.Duration
	apiURL     string
	errorIP    string
}

// defaultClient creates a client with default values.
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

	_, err := url.Parse(c.apiURL)
	if err != nil {
		return nil, fmt.Errorf("invalid service address: %s", c.apiURL)
	}

	return c, nil
}

// GetPublicIP retrieves the public IP of this service instance via ipify.com API.
//
//nolint:nonamedreturns
func (c *Client) GetPublicIP(ctx context.Context) (ip string, err error) {
	ctx, cancelTimeout := context.WithTimeout(ctx, c.timeout)
	defer cancelTimeout()

	req, nerr := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL, nil)
	if nerr != nil {
		return c.errorIP, fmt.Errorf("build request: %w", nerr)
	}

	resp, derr := c.httpClient.Do(req)
	if derr != nil {
		return c.errorIP, fmt.Errorf("failed performing ipify request: %w", derr)
	}

	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()

	if resp.StatusCode != http.StatusOK {
		return c.errorIP, fmt.Errorf("unexpected ipify status code: %d", resp.StatusCode)
	}

	body, berr := io.ReadAll(resp.Body)
	if berr != nil {
		return c.errorIP, fmt.Errorf("failed reading response body: %w", berr)
	}

	return string(body), nil
}
