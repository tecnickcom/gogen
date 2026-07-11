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

	"github.com/tecnickcom/nurago/pkg/httpretrier"
	"github.com/tecnickcom/nurago/pkg/httputil"
	"github.com/tecnickcom/nurago/pkg/validator"
)

// Constants for default timeouts.
const (
	defaultTimeout     = 1 * time.Minute
	defaultPingTimeout = 15 * time.Second

	// defaultMaxRetryAfter caps how long a server-provided Retry-After header can
	// stall a retry, so a rate-limited or misconfigured endpoint cannot force a
	// very long wait on a caller that did not set a context deadline.
	defaultMaxRetryAfter = 60 * time.Second

	// maxBodyBytes caps how much of a response body is drained (for keep-alive
	// connection reuse), bounding time if a misconfigured or hostile endpoint
	// returns a very large body.
	maxBodyBytes = 4 << 10 // 4 KiB

	// maxErrBodyBytes caps the response-body snippet included in error messages,
	// keeping errors (and logs) small.
	maxErrBodyBytes = 512
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
	retrier                     *httpretrier.HTTPRetrier
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
	// through redactAPIKey before being returned or logged. The key is inserted
	// verbatim (not percent-escaped) so redactAPIKey can string-match it; keys
	// are therefore assumed URL-path-safe. A key with a reserved character would
	// only make request building fail (still redacted) — it cannot leak.
	customIncidentURLFormat string
	customMetricURLFormat   string
}

// New constructs a Sleuth API client with validation, retry defaults, and URL templates for the provided org.
func New(addr, org, apiKey string, opts ...Option) (*Client, error) {
	baseURL, err := url.ParseRequestURI(addr)
	if err != nil {
		return nil, fmt.Errorf("%w %q: %w", ErrInvalidAddress, addr, err)
	}

	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf("%w %q: missing scheme or host", ErrInvalidAddress, addr)
	}

	if org == "" {
		return nil, ErrEmptyOrg
	}

	if apiKey == "" {
		return nil, ErrEmptyAPIKey
	}

	valid, err := newValidator(
		validator.WithFieldNameTag("json"),
		validator.WithCustomValidationTags(validator.CustomValidationTags()),
		validator.WithErrorTemplates(validator.ErrorTemplates()),
	)
	if err != nil {
		return nil, fmt.Errorf("init validator: %w", err)
	}

	// Build the static endpoint prefixes with URL.JoinPath so a trailing
	// slash in addr cannot produce "//" in the endpoint paths. The dynamic
	// segments are percent-escaped and substituted at request time.
	orgBase := baseURL.JoinPath("deployments", url.PathEscape(org)).String()

	c := &Client{
		pingTimeout:                 defaultPingTimeout,
		timeout:                     defaultTimeout,
		retryAttempts:               httpretrier.DefaultAttempts,
		retryDelay:                  httpretrier.DefaultDelay,
		apiKey:                      apiKey,
		pingURL:                     orgBase + "/-/register_deploy",
		deployRegistrationURLFormat: orgBase + "/%s/register_deploy",
		manualChangeURLFormat:       orgBase + "/%s/register_manual_deploy",
		customIncidentURLFormat:     orgBase + "/%s/%s/%s/register_impact/%s",
		customMetricURLFormat:       baseURL.JoinPath("impact").String() + "/%d/register_impact",
		valid:                       valid,
	}

	for _, applyOpt := range opts {
		applyOpt(c)
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: c.timeout}
	}

	// The retrier holds only immutable configuration, so a single instance is
	// shared by all requests; building it here surfaces invalid retry settings at
	// construction time instead of on every send. Retry-After is honored (and
	// capped) so a 429 backs off for at least the server-requested time.
	c.retrier, err = httpretrier.New(
		c.httpClient,
		httpretrier.WithRetryIfFn(httpretrier.RetryIfForWriteRequests),
		httpretrier.WithAttempts(c.retryAttempts),
		httpretrier.WithDelay(c.retryDelay),
		httpretrier.WithRespectRetryAfter(),
		httpretrier.WithMaxRetryAfter(defaultMaxRetryAfter),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidRetryConfig, err)
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

	// Drain the body (capped) to allow connection reuse and to surface read
	// errors, without allocating an unbounded buffer for a large body.
	_, rerr := io.Copy(io.Discard, io.LimitReader(resp.Body, maxBodyBytes))
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
		// URL parse errors from http.NewRequestWithContext quote the full URL,
		// which embeds the API key for some endpoints (see the comment on
		// customIncidentURLFormat), so redact it before returning.
		return c.redactAPIKey(rerr)
	}

	resp, derr := c.retrier.Do(r)
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
		// Include a capped snippet of the response body: Sleuth reports the reason
		// for a rejection there, which aids debugging. The body is passed through
		// redactAPIKey defensively in case it ever echoes the key.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrBodyBytes))
		if body = bytes.TrimSpace(body); len(body) > 0 {
			return c.redactAPIKey(fmt.Errorf("sleuth client error - Code: %v, Status: %v, Body: %s", resp.StatusCode, resp.Status, body))
		}

		return c.redactAPIKey(fmt.Errorf("sleuth client error - Code: %v, Status: %v", resp.StatusCode, resp.Status))
	}

	// Drain (capped) so the connection can be reused by keep-alive.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxBodyBytes))

	return nil
}

// SendDeployRegistration registers a deployment event with Sleuth.
func (c *Client) SendDeployRegistration(ctx context.Context, request *DeployRegistrationRequest) error {
	if request == nil {
		return ErrNilRequest
	}

	urlStr := fmt.Sprintf(c.deployRegistrationURLFormat, url.PathEscape(request.Deployment))

	return sendRequest[DeployRegistrationRequest](ctx, c, urlStr, request)
}

// SendManualChange registers a manual change not tracked by source-control-based integrations.
func (c *Client) SendManualChange(ctx context.Context, request *ManualChangeRequest) error {
	if request == nil {
		return ErrNilRequest
	}

	urlStr := fmt.Sprintf(c.manualChangeURLFormat, url.PathEscape(request.Project))

	return sendRequest[ManualChangeRequest](ctx, c, urlStr, request)
}

// SendCustomIncidentImpactRegistration submits custom incident impact values used by Sleuth failure-rate and MTTR metrics.
// The dynamic path segments are percent-escaped so values containing "/", "?",
// or "#" cannot rewrite the request path or shift the API-key segment.
func (c *Client) SendCustomIncidentImpactRegistration(ctx context.Context, request *CustomIncidentImpactRegistrationRequest) error {
	if request == nil {
		return ErrNilRequest
	}

	urlStr := fmt.Sprintf(
		c.customIncidentURLFormat,
		url.PathEscape(request.Project),
		url.PathEscape(request.Environment),
		url.PathEscape(request.ImpactSource),
		// The API key is client configuration (not request data) and is kept
		// verbatim so redactAPIKey can match it in wrapped error messages.
		c.apiKey,
	)

	return sendRequest[CustomIncidentImpactRegistrationRequest](ctx, c, urlStr, request)
}

// SendCustomMetricImpactRegistration submits custom metric impact values for Sleuth anomaly detection and deployment health.
func (c *Client) SendCustomMetricImpactRegistration(ctx context.Context, request *CustomMetricImpactRegistrationRequest) error {
	if request == nil {
		return ErrNilRequest
	}

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
