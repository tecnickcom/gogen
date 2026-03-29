package devlake

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
)

// HTTPClient is the minimal HTTP transport contract used by [Client].
type HTTPClient interface {
	// Do executes the HTTP request.
	Do(req *http.Request) (*http.Response, error)
}

// Client sends deployment and incident webhook events to DevLake.
type Client struct {
	httpClient             HTTPClient
	baseURL                *url.URL
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
// It solves repetitive integration setup by centralizing base URL parsing,
// bearer-token authentication, payload validation, endpoint URL construction,
// and default network timeouts/retry policy.
//
// Example addr: "https://app.devlake.invalid".
func New(addr, apiKey string, opts ...Option) (*Client, error) {
	baseURL, err := url.Parse(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse addr: %w", err)
	}

	if apiKey == "" {
		return nil, errors.New("apiKey is empty")
	}

	valid, _ := validator.New(
		validator.WithFieldNameTag("json"),
		validator.WithCustomValidationTags(validator.CustomValidationTags()),
		validator.WithErrorTemplates(validator.ErrorTemplates()),
	)

	c := &Client{
		baseURL:                baseURL,
		valid:                  valid,
		timeout:                defaultTimeout,
		pingTimeout:            defaultPingTimeout,
		retryDelay:             httpretrier.DefaultDelay,
		retryAttempts:          httpretrier.DefaultAttempts,
		apiKey:                 apiKey,
		pingURL:                fmt.Sprintf("%s/api/rest/version", baseURL),
		deploymentRegURLFormat: fmt.Sprintf("%s/api/rest/plugins/webhook/connections/%%d/deployments", baseURL),
		incidentRegURLFormat:   fmt.Sprintf("%s/api/rest/plugins/webhook/%%d/issues", baseURL),
		incidentCloseURLFormat: fmt.Sprintf("%s/api/rest/plugins/webhook/%%d/issue/%%s/close", baseURL),
	}

	for _, applyOpt := range opts {
		applyOpt(c)
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: c.timeout}
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

	httputil.AddJsonHeaders(r)
	httputil.AddBearerToken(apiKey, r)

	return r, nil
}

// sendRequest validates, retries, and submits a webhook payload to DevLake.
//
// The function performs struct validation, creates a JSON POST request, applies
// write-safe retry behavior, and enforces a successful HTTP 200 response.
func sendRequest[T requestData](ctx context.Context, c *Client, urlStr string, request *T) (err error) {
	err = c.valid.ValidateStructCtx(ctx, request)
	if err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}

	var r *http.Request

	r, err = httpPostRequest(ctx, urlStr, c.apiKey, request)
	if err != nil {
		return err
	}

	var hr *httpretrier.HTTPRetrier

	hr, err = c.newWriteHTTPRetrier()
	if err != nil {
		return fmt.Errorf("create retrier: %w", err)
	}

	var resp *http.Response

	resp, err = hr.Do(r)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}

	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("devlake client error - Code: %v, Status: %v", resp.StatusCode, resp.Status)
	}

	return nil
}

// SendDeployment submits a deployment event to DevLake.
//
// It validates request and posts it to the deployment webhook endpoint scoped
// by ConnectionID.
func (c *Client) SendDeployment(ctx context.Context, request *DeploymentRequest) error {
	urlStr := fmt.Sprintf(c.deploymentRegURLFormat, request.ConnectionID)
	return sendRequest[DeploymentRequest](ctx, c, urlStr, request)
}

// SendIncident submits an incident event to DevLake.
//
// It validates request and posts it to the incident webhook endpoint scoped by
// ConnectionID.
func (c *Client) SendIncident(ctx context.Context, request *IncidentRequest) error {
	urlStr := fmt.Sprintf(c.incidentRegURLFormat, request.ConnectionID)
	return sendRequest[IncidentRequest](ctx, c, urlStr, request)
}

// SendIncidentClose sends an incident-close event for an existing incident.
//
// It validates the close request and calls the dedicated close endpoint using
// ConnectionID and IssueKey.
func (c *Client) SendIncidentClose(ctx context.Context, request *IncidentRequestClose) error {
	err := c.valid.ValidateStructCtx(ctx, request)
	if err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}

	urlStr := fmt.Sprintf(c.incidentCloseURLFormat, request.ConnectionID, request.IssueKey)

	return sendRequest[IncidentRequest](ctx, c, urlStr, nil)
}

// newWriteHTTPRetrier creates the retrier used by write webhook requests.
//
// It applies write-safe retry rules and the configured attempt count.
func (c *Client) newWriteHTTPRetrier() (*httpretrier.HTTPRetrier, error) {
	//nolint:wrapcheck
	return httpretrier.New(
		c.httpClient,
		httpretrier.WithRetryIfFn(httpretrier.RetryIfForWriteRequests),
		httpretrier.WithAttempts(c.retryAttempts),
	)
}
