package devlake

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

	"github.com/tecnickcom/nurago/pkg/httpretrier"
	"github.com/tecnickcom/nurago/pkg/httputil"
	"github.com/tecnickcom/nurago/pkg/validator"
)

// Default configuration values.
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

// HTTPClient is the minimal HTTP transport contract used by [Client].
type HTTPClient interface {
	// Do executes the HTTP request.
	Do(req *http.Request) (*http.Response, error)
}

// newValidator is a package-level indirection so tests can force a
// validator initialization failure; in production it always delegates
// to [validator.New].
var newValidator = validator.New //nolint:gochecknoglobals

// Client sends deployment and incident webhook events to DevLake.
type Client struct {
	httpClient             HTTPClient
	retrier                *httpretrier.HTTPRetrier
	valid                  *validator.Validator
	timeout                time.Duration
	pingTimeout            time.Duration
	retryDelay             time.Duration
	retryAttempts          uint
	apiKey                 string
	pingURL                string
	deploymentRegURLFormat string
	incidentRegURLFormat   string
	incidentCloseURLFormat string
}

// New creates a DevLake webhook client with validation and retry defaults.
//
// It parses the base URL and configures bearer-token authentication, payload
// validation, endpoint URL construction, and default network timeouts and
// retry policy.
//
// Example addr: "https://app.devlake.invalid".
func New(addr, apiKey string, opts ...Option) (*Client, error) {
	baseURL, err := url.ParseRequestURI(addr)
	if err != nil {
		return nil, fmt.Errorf("%w %q: %w", ErrInvalidAddress, addr, err)
	}

	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf("%w %q: missing scheme or host", ErrInvalidAddress, addr)
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

	// Build the static endpoint prefix with URL.JoinPath so a trailing slash
	// in addr cannot produce "//" in the endpoint paths (strict routers treat
	// "//" as a different, 404 path).
	apiBase := baseURL.JoinPath("api", "rest").String()

	c := &Client{
		valid:                  valid,
		timeout:                defaultTimeout,
		pingTimeout:            defaultPingTimeout,
		retryDelay:             httpretrier.DefaultDelay,
		retryAttempts:          httpretrier.DefaultAttempts,
		apiKey:                 apiKey,
		pingURL:                apiBase + "/version",
		deploymentRegURLFormat: apiBase + "/plugins/webhook/connections/%d/deployments",
		// NOTE: the DevLake webhook plugin registers each endpoint under both a legacy
		// ".../webhook/:connectionId/..." form and a newer ".../webhook/connections/:connectionId/..."
		// form. All three URLs below are valid registered routes. The deployment URL uses
		// the "connections/" form (requires a recent DevLake), while the incident URLs use
		// the legacy form (also valid on older DevLake); the close path uses singular
		// "issue". See https://devlake.apache.org/docs/Plugins/webhook/
		incidentRegURLFormat:   apiBase + "/plugins/webhook/%d/issues",
		incidentCloseURLFormat: apiBase + "/plugins/webhook/%d/issue/%s/close",
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

// HealthCheck verifies that the DevLake API endpoint is reachable and healthy.
//
// The request uses pingTimeout and returns an error for transport failures,
// timeout failures, or non-200 responses.
func (c *Client) HealthCheck(ctx context.Context) (err error) {
	ctx, cancel := context.WithTimeout(ctx, c.pingTimeout)
	defer cancel()

	req, nerr := http.NewRequestWithContext(ctx, http.MethodGet, c.pingURL, nil)
	if nerr != nil {
		return fmt.Errorf("create get request: %w", nerr)
	}

	httputil.AddBearerToken(c.apiKey, req)
	req.Header.Set(httputil.HeaderAccept, httputil.MimeTypeJSON)

	resp, derr := c.httpClient.Do(req)
	if derr != nil {
		return fmt.Errorf("healthcheck request: %w", derr)
	}

	defer func() {
		// Drain (capped) so the connection can be reused by keep-alive.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxBodyBytes))
		err = errors.Join(err, resp.Body.Close())
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected healthcheck status code: %d", resp.StatusCode)
	}

	return nil
}

// httpPostRequest builds an authenticated JSON POST request.
//
// It encodes request as JSON and attaches standard JSON and bearer-token
// headers expected by the DevLake webhook API.
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
	httputil.AddBearerToken(apiKey, r)

	return r, nil
}

// sendRequest validates, retries, and submits a webhook payload to DevLake.
//
// The function performs struct validation, creates a JSON POST request, applies
// write-safe retry behavior, and enforces a successful HTTP 200 response.
func sendRequest[T requestData](ctx context.Context, c *Client, urlStr string, request *T) error {
	err := c.valid.ValidateStructCtx(ctx, request)
	if err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}

	r, err := httpPostRequest(ctx, urlStr, c.apiKey, request)
	if err != nil {
		return err
	}

	return c.doRequest(r)
}

// SendDeployment submits a deployment event to DevLake.
//
// It validates request and posts it to the deployment webhook endpoint scoped
// by ConnectionID.
func (c *Client) SendDeployment(ctx context.Context, request *DeploymentRequest) error {
	if request == nil {
		return ErrNilRequest
	}

	urlStr := fmt.Sprintf(c.deploymentRegURLFormat, request.ConnectionID)

	return sendRequest[DeploymentRequest](ctx, c, urlStr, request)
}

// SendIncident submits an incident event to DevLake.
//
// It validates request and posts it to the incident webhook endpoint scoped by
// ConnectionID.
func (c *Client) SendIncident(ctx context.Context, request *IncidentRequest) error {
	if request == nil {
		return ErrNilRequest
	}

	urlStr := fmt.Sprintf(c.incidentRegURLFormat, request.ConnectionID)

	return sendRequest[IncidentRequest](ctx, c, urlStr, request)
}

// SendIncidentClose sends an incident-close event for an existing incident.
//
// It validates the close request and calls the dedicated close endpoint using
// ConnectionID and IssueKey.
func (c *Client) SendIncidentClose(ctx context.Context, request *IncidentRequestClose) error {
	if request == nil {
		return ErrNilRequest
	}

	err := c.valid.ValidateStructCtx(ctx, request)
	if err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}

	// Percent-escape the issue key (it may come from an external tracker) so
	// values containing "/", "?", or "#" cannot rewrite the request path or
	// swallow the trailing "/close" segment.
	urlStr := fmt.Sprintf(c.incidentCloseURLFormat, request.ConnectionID, url.PathEscape(request.IssueKey))

	// The close endpoint takes no payload, so post an empty body. Reusing the
	// generic sendRequest with a typed-nil payload would JSON-encode the literal
	// "null" and bypass validation (go-playground/validator returns an
	// InvalidValidationError for a nil pointer, not ValidationErrors).
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, http.NoBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httputil.AddJSONHeaders(r)
	httputil.AddBearerToken(c.apiKey, r)

	return c.doRequest(r)
}

// doRequest applies write-safe retry behavior and enforces an HTTP 200 response.
//
// It is shared by payload-carrying webhook calls and the bodyless close call.
func (c *Client) doRequest(r *http.Request) (err error) {
	resp, err := c.retrier.Do(r)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}

	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()

	if resp.StatusCode != http.StatusOK {
		// Include a capped snippet of the response body: the DevLake webhook API
		// reports the reason for a rejection there, which aids debugging.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrBodyBytes))
		if body = bytes.TrimSpace(body); len(body) > 0 {
			return fmt.Errorf("devlake client error - Code: %v, Status: %v, Body: %s", resp.StatusCode, resp.Status, body)
		}

		return fmt.Errorf("devlake client error - Code: %v, Status: %v", resp.StatusCode, resp.Status)
	}

	// Drain (capped) so the connection can be reused by keep-alive.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxBodyBytes))

	return nil
}
