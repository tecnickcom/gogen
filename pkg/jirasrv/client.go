package jirasrv

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/tecnickcom/gogen/pkg/httpretrier"
	"github.com/tecnickcom/gogen/pkg/httputil"
	"github.com/tecnickcom/gogen/pkg/validator"
)

// Default configuration values.
const (
	defaultTimeout     = 1 * time.Minute
	defaultPingTimeout = 15 * time.Second
	apiBasePath        = "/rest/api/2" // https://docs.atlassian.com/software/jira/docs/api/REST/9.17.0/
)

// HTTPClient is the minimal HTTP transport contract used by [Client].
type HTTPClient interface {
	// Do sends an HTTP request and returns an HTTP response.
	Do(req *http.Request) (*http.Response, error)
}

// Client wraps Jira Server REST API operations with validation and retries.
type Client struct {
	httpClient    HTTPClient
	baseURL       *url.URL
	apiURL        *url.URL
	valid         *validator.Validator
	timeout       time.Duration
	pingTimeout   time.Duration
	retryDelay    time.Duration
	retryAttempts uint
	token         string
	pingAddr      string
}

// New constructs a Jira Server REST client.
//
// It validates base URL and token, initializes validators and endpoint paths,
// applies options, and provisions a default HTTP client when none is provided.
func New(addr, token string, opts ...Option) (*Client, error) {
	baseURL, err := url.Parse(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse addr: %w", err)
	}

	if token == "" {
		return nil, errors.New("token is empty")
	}

	valid, _ := validator.New(
		validator.WithFieldNameTag("json"),
		validator.WithCustomValidationTags(validator.CustomValidationTags()),
		validator.WithErrorTemplates(validator.ErrorTemplates()),
	)

	apiURL := baseURL.JoinPath(apiBasePath)

	c := &Client{
		baseURL:       baseURL,
		apiURL:        apiURL,
		valid:         valid,
		timeout:       defaultTimeout,
		pingTimeout:   defaultPingTimeout,
		retryDelay:    httpretrier.DefaultDelay,
		retryAttempts: httpretrier.DefaultAttempts,
		token:         token,
		pingAddr:      apiURL.JoinPath("serverInfo").String() + "?doHealthCheck=true",
	}

	for _, applyOpt := range opts {
		applyOpt(c)
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: c.timeout}
	}

	return c, nil
}

// HealthCheck verifies Jira server reachability and health endpoint status.
//
// It runs with pingTimeout and returns an error for transport failures or
// unexpected HTTP status codes.
func (c *Client) HealthCheck(ctx context.Context) (err error) {
	ctx, cancel := context.WithTimeout(ctx, c.pingTimeout)
	defer cancel()

	req, rerr := c.httpRequest(ctx, http.MethodGet, c.pingAddr, nil)
	if rerr != nil {
		return rerr
	}

	resp, derr := c.httpClient.Do(req)
	if derr != nil {
		return fmt.Errorf("healthcheck request: %w", derr)
	}

	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected healthcheck status code: %d", resp.StatusCode)
	}

	return nil
}

// SendRequest validates and sends a Jira REST API request.
//
// It builds the target URL from endpoint and query, JSON-encodes request when
// provided, applies auth/content headers, and executes through the configured
// method-aware retrier.
// NOTE: The caller must close the response body.
func (c *Client) SendRequest(
	ctx context.Context,
	httpMethod string,
	endpoint string,
	query *url.Values,
	request any,
) (*http.Response, error) {
	var (
		buffer io.Reader
		err    error
	)

	if request != nil {
		buffer, err = c.requestBuffer(ctx, request)
		if err != nil {
			return nil, err
		}
	}

	targetURL := c.apiURL.JoinPath(endpoint)
	if query != nil {
		targetURL.RawQuery = query.Encode()
	}

	r, err := c.httpRequest(ctx, httpMethod, targetURL.String(), buffer)
	if err != nil {
		return nil, err
	}

	hr, err := c.newHTTPRetrier(httpMethod)
	if err != nil {
		return nil, fmt.Errorf("create retrier: %w", err)
	}

	return hr.Do(r) //nolint:wrapcheck
}

// requestBuffer validates request and JSON-encodes it into a buffer.
//
// This keeps SendRequest focused on transport flow while enforcing payload
// schema checks before network calls.
func (c *Client) requestBuffer(
	ctx context.Context,
	request any,
) (*bytes.Buffer, error) {
	err := c.valid.ValidateStructCtx(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	buf := &bytes.Buffer{}

	err = json.NewEncoder(buf).Encode(request)
	if err != nil {
		return nil, fmt.Errorf("json encoding: %w", err)
	}

	return buf, nil
}

// httpRequest builds an HTTP request with Jira default headers attached.
func (c *Client) httpRequest(
	ctx context.Context,
	httpMethod string,
	urlStr string,
	request io.Reader,
) (*http.Request, error) {
	r, err := http.NewRequestWithContext(ctx, httpMethod, urlStr, request)
	if err != nil {
		return nil, fmt.Errorf("create http request: %w", err)
	}

	c.setRequestHeaders(r)

	return r, nil
}

// setRequestHeaders attaches JSON and bearer-token headers to r.
func (c *Client) setRequestHeaders(r *http.Request) {
	httputil.AddJsonHeaders(r)
	httputil.AddBearerToken(c.token, r)
}

// newHTTPRetrier creates a retrier configured for the given HTTP method.
//
// Retry behavior follows method semantics (for example write methods are
// retried more conservatively).
func (c *Client) newHTTPRetrier(httpMethod string) (*httpretrier.HTTPRetrier, error) {
	//nolint:wrapcheck
	return httpretrier.New(
		c.httpClient,
		httpretrier.WithRetryIfFn(httpretrier.RetryIfFnByHTTPMethod(httpMethod)),
		httpretrier.WithAttempts(c.retryAttempts),
	)
}
