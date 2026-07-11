package ipify

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Default configuration values.
const (
	defaultTimeout = 4 * time.Second         // default request timeout
	defaultAPIURL  = "https://api.ipify.org" // use "https://api64.ipify.org" for IPv6 support
	defaultErrorIP = ""                      // string to return in case of error in place of the IP

	// maxBodyBytes caps how much of the response body is read. The endpoint is
	// caller-configurable via WithURL, and a valid answer is tiny (the longest
	// textual IPv6 form is 45 bytes), so this bounds memory use if a
	// misconfigured or hostile endpoint returns a large body.
	maxBodyBytes = 64
)

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
// validates the configured API URL. A non-positive timeout is silently clamped
// to the default rather than rejected, so a misconfigured timeout cannot make
// every request fail with an already-expired context. An API URL that is
// missing, unparseable, or that does not use an http/https scheme with a host
// is rejected with an error matching [ErrInvalidOptions].
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
		return nil, fmt.Errorf("invalid ipify service address %q: %w: %w", c.apiURL, ErrInvalidOptions, err)
	}

	// http.Client only speaks http/https; any other scheme would fail at request
	// time, so reject it up front with a clear construction-time error.
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("invalid ipify service address %q (scheme must be http or https): %w", c.apiURL, ErrInvalidOptions)
	}

	if parsed.Host == "" {
		return nil, fmt.Errorf("invalid ipify service address %q (missing host): %w", c.apiURL, ErrInvalidOptions)
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

	// A conforming http.Client never returns a nil body on a nil error, but the
	// transport is caller-supplied via WithHTTPClient; guard against a
	// misbehaving implementation so the deferred Close and the body reads below
	// cannot panic with a nil-pointer dereference.
	if resp.Body == nil {
		return c.errorIP, fmt.Errorf("nil response body: %w", ErrInvalidResponse)
	}

	defer func() {
		err = errors.Join(err, resp.Body.Close())
		if err != nil {
			// Honor the documented contract: on any failure — including a
			// close error after a successful read — return the configured
			// fallback errorIP together with the error.
			ip = c.errorIP
		}
	}()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxBodyBytes))

		return c.errorIP, fmt.Errorf("unexpected ipify status code: %d", resp.StatusCode)
	}

	body, berr := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if berr != nil {
		return c.errorIP, fmt.Errorf("failed reading response body: %w", berr)
	}

	ip = strings.TrimSpace(string(body))
	if ip == "" {
		return c.errorIP, fmt.Errorf("empty ipify response: %w", ErrInvalidResponse)
	}

	return ip, nil
}

// HealthCheck verifies that the configured ipify endpoint is reachable and
// returns a usable public IP.
//
// It performs a GetPublicIP call, discards the resolved address, and returns
// only the error. It exists for parity with the other nurago HTTP clients that
// expose a HealthCheck probe.
func (c *Client) HealthCheck(ctx context.Context) error {
	_, err := c.GetPublicIP(ctx)

	return err
}
