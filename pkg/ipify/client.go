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

// errMissingSchemeOrHost is returned when the configured API URL lacks a scheme or host.
var errMissingSchemeOrHost = errors.New("missing scheme or host")

// HTTPClient is the minimal HTTP transport contract used by [Client].
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client resolves public IP addresses through an ipify-compatible endpoint.
type Client struct {
	httpClient HTTPClient
	timeout    time.Duration
	apiURL     string
	errorIP    string
}

// defaultClient returns a Client preloaded with package defaults.
//
// It keeps New focused on option application and validation.
func defaultClient() *Client {
	return &Client{
		timeout: defaultTimeout,
		apiURL:  defaultAPIURL,
		errorIP: defaultErrorIP,
	}
}

// New constructs an ipify client with validated configuration.
//
// It applies options, initializes a default HTTP client when needed, and
// validates the configured API URL.
func New(opts ...Option) (*Client, error) {
	c := defaultClient()

	for _, applyOpt := range opts {
		applyOpt(c)
	}

	// Guard against a non-positive timeout, which would produce an
	// already-expired context and make every request fail.
	if c.timeout <= 0 {
		c.timeout = defaultTimeout
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: c.timeout}
	}

	parsed, err := url.ParseRequestURI(c.apiURL)
	if err != nil {
		return nil, fmt.Errorf("invalid ipify service address %q: %w", c.apiURL, err)
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid ipify service address %q: %w", c.apiURL, errMissingSchemeOrHost)
	}

	return c, nil
}

// GetPublicIP resolves the instance public IP through the configured ipify endpoint.
//
// On any request or response failure, it returns the configured fallback
// errorIP together with the error.
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
		_, _ = io.Copy(io.Discard, resp.Body)

		return c.errorIP, fmt.Errorf("unexpected ipify status code: %d", resp.StatusCode)
	}

	body, berr := io.ReadAll(resp.Body)
	if berr != nil {
		return c.errorIP, fmt.Errorf("failed reading response body: %w", berr)
	}

	return string(body), nil
}
