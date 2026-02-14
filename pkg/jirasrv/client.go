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

// HTTPClient contains the function to perform the actual HTTP request.
type HTTPClient interface {
	// Do sends an HTTP request and returns an HTTP response.
	Do(req *http.Request) (*http.Response, error)
}

// Client represents the config options required by this client.
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

// New creates a new client instance.
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

// HealthCheck performs a status check on this service.
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

// SendRequest sends an HTTP request to the Jira server and returns the response.
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

// requestBuffer validate the request and returns the data as buffer.
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

// httpRequest prepares a generic HTTP request.
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

// setRequestHeaders sets the required headers on the request.
func (c *Client) setRequestHeaders(r *http.Request) {
	httputil.AddJsonHeaders(r)
	httputil.AddBearerToken(c.token, r)
}

// newHTTPRetrier creates a new HTTP retrier instance.
func (c *Client) newHTTPRetrier(httpMethod string) (*httpretrier.HTTPRetrier, error) {
	//nolint:wrapcheck
	return httpretrier.New(
		c.httpClient,
		httpretrier.WithRetryIfFn(httpretrier.RetryIfFnByHTTPMethod(httpMethod)),
		httpretrier.WithAttempts(c.retryAttempts),
	)
}
