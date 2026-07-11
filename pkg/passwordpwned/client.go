package passwordpwned

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tecnickcom/nurago/pkg/httpretrier"
)

const (
	defaultTimeout   = 30 * time.Second
	defaultAPIURL    = "https://api.pwnedpasswords.com"
	rangePath        = "range"
	defaultUserAgent = "nurago.passwordpwned/1"

	// defaultMaxResponseBytes bounds the decoded response size to guard against
	// decompression-bomb style memory exhaustion. Real HIBP range responses are
	// ~30 KB even fully padded, so this leaves generous headroom.
	defaultMaxResponseBytes = 8 << 20 // 8 MiB

	// defaultMaxRetryAfter caps how long a server-provided Retry-After header can
	// stall a retry, so a hostile or misconfigured endpoint cannot force a very
	// long wait on a caller that did not set a context deadline.
	defaultMaxRetryAfter = 60 * time.Second

	// defaultPingTimeout bounds the [Client.HealthCheck] probe request.
	defaultPingTimeout = 5 * time.Second

	// healthCheckPrefix is a fixed, valid hash prefix used by the HealthCheck
	// probe: every 5-hex-char range exists, so a healthy endpoint returns 200.
	healthCheckPrefix = "00000"
)

// HTTPClient is the minimal HTTP transport used by [Client].
type HTTPClient interface {
	// Do sends an HTTP request and returns an HTTP response.
	Do(req *http.Request) (*http.Response, error)
}

// Client queries the HIBP Pwned Passwords API using the k-Anonymity range endpoint.
// Create one with [New]; the zero value is not usable.
type Client struct {
	httpClient       HTTPClient
	retrier          *httpretrier.HTTPRetrier
	timeout          time.Duration
	pingTimeout      time.Duration
	retryDelay       time.Duration
	retryAttempts    uint
	apiURL           string
	userAgent        string
	maxResponseBytes int64
}

// defaultClient returns a client preconfigured with package defaults.
func defaultClient() *Client {
	return &Client{
		timeout:          defaultTimeout,
		pingTimeout:      defaultPingTimeout,
		retryAttempts:    httpretrier.DefaultAttempts,
		retryDelay:       httpretrier.DefaultDelay,
		apiURL:           defaultAPIURL,
		userAgent:        defaultUserAgent,
		maxResponseBytes: defaultMaxResponseBytes,
	}
}

// New creates a [Client] with the given options applied over sensible defaults.
//
// Defaults:
//   - API URL:        https://api.pwnedpasswords.com
//   - Timeout:        30 s
//   - Ping timeout:   5 s (HealthCheck probe)
//   - User-Agent:     nurago.passwordpwned/1
//   - Retry attempts: httpretrier.DefaultAttempts
//   - Max response:   8 MiB (decoded)
//
// Use [WithURL], [WithTimeout], [WithHTTPClient], [WithUserAgent],
// [WithRetryAttempts], [WithRetryDelay], [WithResponseSizeLimit], or
// [WithPingTimeout] to override individual settings.
//
// All configuration is validated here: an invalid URL, an invalid User-Agent,
// or invalid retry settings return an error, so a successfully constructed
// Client cannot fail on configuration at call time.
func New(opts ...Option) (*Client, error) {
	c := defaultClient()

	for _, applyOpt := range opts {
		applyOpt(c)
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: c.timeout}
	}

	uaErr := validateUserAgent(c.userAgent)
	if uaErr != nil {
		return nil, uaErr
	}

	u, err := url.ParseRequestURI(c.apiURL)
	if err != nil {
		return nil, fmt.Errorf("%w %q: %w", ErrInvalidURL, c.apiURL, err)
	}

	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("%w %q: missing scheme or host", ErrInvalidURL, c.apiURL)
	}

	// A query or fragment would end up between the host and the appended
	// "/range/<prefix>" path, silently building a garbage request URL.
	if u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("%w %q: must not contain a query or fragment", ErrInvalidURL, c.apiURL)
	}

	// Normalize away any trailing slashes so request paths are built cleanly.
	c.apiURL = strings.TrimRight(c.apiURL, "/")

	// The retrier holds only immutable configuration, so a single instance is
	// shared by all calls; building it here surfaces invalid retry settings at
	// construction time instead of on first use.
	c.retrier, err = httpretrier.New(
		c.httpClient,
		httpretrier.WithRetryIfFn(httpretrier.RetryIfForReadRequests),
		httpretrier.WithAttempts(c.retryAttempts),
		httpretrier.WithDelay(c.retryDelay),
		httpretrier.WithRespectRetryAfter(),
		httpretrier.WithMaxRetryAfter(defaultMaxRetryAfter),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid retry configuration: %w", err)
	}

	return c, nil
}

// validateUserAgent rejects an empty value (Go would suppress the header
// entirely and the HIBP API requires one) and control bytes (the HTTP
// transport rejects invalid header field values on every request), so both
// misconfigurations fail at construction instead of at call time.
func validateUserAgent(s string) error {
	if s == "" {
		return fmt.Errorf("%w: must not be empty", ErrInvalidUserAgent)
	}

	for _, r := range s {
		if r < 0x20 || r == 0x7F {
			return fmt.Errorf("%w: must not contain control characters", ErrInvalidUserAgent)
		}
	}

	return nil
}

// HealthCheck verifies that the configured Pwned Passwords endpoint is
// reachable and healthy.
//
// The HIBP range API has no dedicated ping endpoint, so the probe performs a
// lightweight GET on a fixed range prefix (without the Add-Padding header, to
// keep the response small) and reports any transport failure or non-200 status.
// The response body is not decoded: the status code is the readiness signal.
// The request is bounded by the ping timeout (see [WithPingTimeout]) and is
// never retried.
//
// Note: each probe is a live request against the configured endpoint; the HIBP
// range API is unauthenticated and CDN-cached, so the cost is negligible at
// normal probe cadences.
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.pingTimeout)
	defer cancel()

	r, nerr := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL+"/"+rangePath+"/"+healthCheckPrefix, nil)
	if nerr != nil {
		return fmt.Errorf("create healthcheck request: %w", nerr)
	}

	r.Header.Set("User-Agent", c.userAgent)

	resp, derr := c.httpClient.Do(r)
	if derr != nil {
		return fmt.Errorf("healthcheck request: %w", derr)
	}

	// A nil body from a non-conforming injected HTTPClient must not panic; the
	// status code below is still a valid readiness signal.
	if resp.Body != nil {
		// Drain (bounded) and close so the connection can be reused; the close
		// error is intentionally ignored, as the status is already in hand.
		defer func() {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, c.maxResponseBytes))
			_ = resp.Body.Close()
		}()
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck: %w: %d", ErrUnexpectedStatus, resp.StatusCode)
	}

	return nil
}
