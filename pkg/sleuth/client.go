package sleuth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tecnickcom/gogen/pkg/httpretrier"
	"github.com/tecnickcom/gogen/pkg/httputil"
	"github.com/tecnickcom/gogen/pkg/validator"
)

// Constants for default timeouts.
const (
	defaultTimeout     = 1 * time.Minute
	defaultPingTimeout = 15 * time.Second
)

// newValidator is a package-level indirection over [validator.New] so tests can
// force an initialization failure; in production it always delegates to validator.New.
var newValidator = validator.New //nolint:gochecknoglobals

// HTTPClient is the minimal HTTP transport contract used by [Client].
type HTTPClient interface {
	// Do sends an HTTP request and returns an HTTP response.
	Do(req *http.Request) (*http.Response, error)
}

// Client sends deployment/change/impact events to the Sleuth API.
type Client struct {
	httpClient                  HTTPClient
	baseURL                     *url.URL
	valid                       *validator.Validator
	timeout                     time.Duration
	pingTimeout                 time.Duration
	retryDelay                  time.Duration
	retryAttempts               uint
	apiKey                      string
	pingURL                     string
	deployRegistrationURLFormat string
	manualChangeURLFormat       string
	// customIncidentURLFormat ends in .../register_impact/<apiKey>: the Sleuth
	// API key is part of the request URL path. Because secrets in URLs are
	// prone to leaking (e.g. via *url.Error in wrapped transport errors), any
	// error surfaced from a request built with this format must be passed
	// through redactAPIKey before being returned or logged.
	customIncidentURLFormat string
	customMetricURLFormat   string
}

// New constructs a Sleuth API client with validation, retry defaults, and URL templates for the provided org.
func New(addr, org, apiKey string, opts ...Option) (*Client, error) {
	baseURL, err := url.ParseRequestURI(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid sleuth address %q: %w", addr, err)
	}

	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf("invalid sleuth address %q: missing scheme or host", addr)
	}

	if org == "" {
		return nil, errors.New("org is empty")
	}

	if apiKey == "" {
		return nil, errors.New("apiKey is empty")
	}

	valid, err := newValidator(
		validator.WithFieldNameTag("json"),
		validator.WithCustomValidationTags(validator.CustomValidationTags()),
		validator.WithErrorTemplates(validator.ErrorTemplates()),
	)
	if err != nil {
		return nil, fmt.Errorf("init validator: %w", err)
	}

	c := &Client{
		baseURL:                     baseURL,
		pingTimeout:                 defaultPingTimeout,
		timeout:                     defaultTimeout,
		retryAttempts:               httpretrier.DefaultAttempts,
		retryDelay:                  httpretrier.DefaultDelay,
		apiKey:                      apiKey,
		pingURL:                     fmt.Sprintf("%s/deployments/%s/-/register_deploy", baseURL, org),
		deployRegistrationURLFormat: fmt.Sprintf("%s/deployments/%s/%%s/register_deploy", baseURL, org),
		manualChangeURLFormat:       fmt.Sprintf("%s/deployments/%s/%%s/register_manual_deploy", baseURL, org),
		customIncidentURLFormat:     fmt.Sprintf("%s/deployments/%s/%%s/%%s/%%s/register_impact/%%s", baseURL, org),
		customMetricURLFormat:       fmt.Sprintf("%s/impact/%%d/register_impact", baseURL),
		valid:                       valid,
	}

	for _, applyOpt := range opts {
		applyOpt(c)
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: c.timeout}
	}

	return c, nil
}

// HealthCheck validates API access by executing a controlled request and verifying Sleuth's expected 404 response pattern.
func (c *Client) HealthCheck(ctx context.Context) (err error) {
	ctx, cancel := context.WithTimeout(ctx, c.pingTimeout)
	defer cancel()

	req, nerr := httpPostRequest(
		ctx,
		c.pingURL,
		c.apiKey,
		&DeployRegistrationRequest{
			Sha:               "0",
			Environment:       "TEST",
			IgnoreIfDuplicate: true,
		},
	)
	if nerr != nil {
		return nerr
	}

	resp, derr := c.httpClient.Do(req)
	if derr != nil {
		// The Sleuth API key is embedded in some request URL paths (see the
		// comment on customIncidentURLFormat), so transport errors (typically
		// *url.Error) may expose it via Error(). Redact it before returning.
		return c.redactAPIKey(fmt.Errorf("healthcheck request: %w", derr))
	}

	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()

	// Sleuth answers the controlled probe deployment with a 404, so the 404
	// status code alone confirms reachable, authenticated API access. We rely on
	// the status code rather than matching a localized response body string,
	// which would be brittle if Sleuth changed its message wording.
	if resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("unexpected healthcheck status code: %d", resp.StatusCode)
	}

	// Drain the body to allow connection reuse and to surface read errors.
	_, rerr := io.ReadAll(resp.Body)
	if rerr != nil {
		return fmt.Errorf("failed reading response body: %w", rerr)
	}

	return nil
}

// httpPostRequest builds an authenticated JSON POST request for Sleuth endpoints.
func httpPostRequest(ctx context.Context, urlStr, apiKey string, request any) (*http.Request, error) {
	buffer := &bytes.Buffer{}

	err := json.NewEncoder(buffer).Encode(request)
	if err != nil {
		return nil, fmt.Errorf("json encoding: %w", err)
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, buffer)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httputil.AddJSONHeaders(r)
	httputil.AddAuthorizationHeader("apikey "+apiKey, r)

	return r, nil
}

// sendRequest validates payload, executes request with retrier, and enforces HTTP 200 success.
func sendRequest[T requestData](ctx context.Context, c *Client, urlStr string, request *T) (err error) {
	verr := c.valid.ValidateStructCtx(ctx, request)
	if verr != nil {
		return fmt.Errorf("invalid request: %w", verr)
	}

	r, rerr := httpPostRequest(ctx, urlStr, c.apiKey, request)
	if rerr != nil {
		return rerr
	}

	hr, herr := c.newWriteHTTPRetrier()
	if herr != nil {
		return fmt.Errorf("create retrier: %w", herr)
	}

	resp, derr := hr.Do(r)
	if derr != nil {
		// Some Sleuth endpoints embed the API key in the request URL path (see
		// the comment on customIncidentURLFormat). Go's HTTP client wraps
		// transport failures in *url.Error, whose Error() includes the full
		// URL and therefore the secret. Redact the key before returning so it
		// cannot leak into logs.
		return c.redactAPIKey(fmt.Errorf("execute request: %w", derr))
	}

	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sleuth client error - Code: %v, Status: %v", resp.StatusCode, resp.Status)
	}

	return nil
}

// SendDeployRegistration registers a deployment event with Sleuth.
func (c *Client) SendDeployRegistration(ctx context.Context, request *DeployRegistrationRequest) error {
	urlStr := fmt.Sprintf(c.deployRegistrationURLFormat, request.Deployment)
	return sendRequest[DeployRegistrationRequest](ctx, c, urlStr, request)
}

// SendManualChange registers a manual change not tracked by source-control-based integrations.
func (c *Client) SendManualChange(ctx context.Context, request *ManualChangeRequest) error {
	urlStr := fmt.Sprintf(c.manualChangeURLFormat, request.Project)
	return sendRequest[ManualChangeRequest](ctx, c, urlStr, request)
}

// SendCustomIncidentImpactRegistration submits custom incident impact values used by Sleuth failure-rate and MTTR metrics.
func (c *Client) SendCustomIncidentImpactRegistration(ctx context.Context, request *CustomIncidentImpactRegistrationRequest) error {
	urlStr := fmt.Sprintf(c.customIncidentURLFormat, request.Project, request.Environment, request.ImpactSource, c.apiKey)
	return sendRequest[CustomIncidentImpactRegistrationRequest](ctx, c, urlStr, request)
}

// SendCustomMetricImpactRegistration submits custom metric impact values for Sleuth anomaly detection and deployment health.
func (c *Client) SendCustomMetricImpactRegistration(ctx context.Context, request *CustomMetricImpactRegistrationRequest) error {
	urlStr := fmt.Sprintf(c.customMetricURLFormat, request.ImpactID)
	return sendRequest[CustomMetricImpactRegistrationRequest](ctx, c, urlStr, request)
}

// redactAPIKey returns an error whose message has every occurrence of the
// client's API key replaced with "REDACTED". This guards against the secret
// leaking through wrapped transport errors: Go's HTTP client returns a
// *url.Error whose Error() string contains the full request URL, and some
// Sleuth endpoints embed the API key in that URL path (see the comment on
// customIncidentURLFormat). The original error is preserved via Unwrap so
// callers can still inspect it with errors.Is / errors.As.
func (c *Client) redactAPIKey(err error) error {
	if err == nil {
		return nil
	}

	if !strings.Contains(err.Error(), c.apiKey) {
		return err
	}

	return &redactedError{
		msg: strings.ReplaceAll(err.Error(), c.apiKey, "REDACTED"),
		err: err,
	}
}

// redactedError wraps an error with a sanitized message while keeping the
// original error reachable through Unwrap for errors.Is / errors.As checks.
type redactedError struct {
	err error
	msg string
}

// Error returns the sanitized error message with the API key redacted.
func (e *redactedError) Error() string { return e.msg }

// Unwrap exposes the original error for errors.Is / errors.As.
func (e *redactedError) Unwrap() error { return e.err }

// newWriteHTTPRetrier creates a write-oriented HTTP retrier using configured attempt count.
func (c *Client) newWriteHTTPRetrier() (*httpretrier.HTTPRetrier, error) {
	//nolint:wrapcheck
	return httpretrier.New(
		c.httpClient,
		httpretrier.WithRetryIfFn(httpretrier.RetryIfForWriteRequests),
		httpretrier.WithAttempts(c.retryAttempts),
		httpretrier.WithDelay(c.retryDelay),
	)
}
