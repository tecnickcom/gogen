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

	"github.com/Vonage/gosrvlib/pkg/httpretrier"
	"github.com/Vonage/gosrvlib/pkg/logging"
	"github.com/Vonage/gosrvlib/pkg/validator"
)

const (
	defaultTimeout      = 1 * time.Minute
	defaultPingTimeout  = 15 * time.Second
	headerAuthorization = "Authorization"
	headerContentType   = "Content-Type"
	headerAccept        = "Accept"
	contentType         = "application/json"
)

// HTTPClient contains the function to perform the actual HTTP request.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client represents the config options required by this client.
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

// New creates a new client instance.
// Example for addr: "https://app.devlake.invalid"
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

// HealthCheck performs a status check on this service.
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.pingTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.pingURL, nil)
	if err != nil {
		return fmt.Errorf("create get request: %w", err)
	}

	req.Header.Set(headerAuthorization, "Bearer "+c.apiKey)
	req.Header.Set(headerAccept, contentType)

	resp, err := c.httpClient.Do(req) //nolint:bodyclose
	if err != nil {
		return fmt.Errorf("healthcheck request: %w", err)
	}

	defer logging.Close(ctx, resp.Body, "error while closing HealthCheck response body")

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected healthcheck status code: %d", resp.StatusCode)
	}

	return nil
}

// httpPostRequest prepare an HTTP request encoding the payload as JSON.
func httpPostRequest(ctx context.Context, urlStr, apiKey string, request any) (*http.Request, error) {
	buffer := &bytes.Buffer{}

	if err := json.NewEncoder(buffer).Encode(request); err != nil {
		return nil, fmt.Errorf("json encoding: %w", err)
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, buffer)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	r.Header.Set(headerAuthorization, "Bearer "+apiKey)
	r.Header.Set(headerContentType, contentType)
	r.Header.Set(headerAccept, contentType)

	return r, nil
}

func (c *Client) newWriteHTTPRetrier() (*httpretrier.HTTPRetrier, error) {
	//nolint:wrapcheck
	return httpretrier.New(
		c.httpClient,
		httpretrier.WithRetryIfFn(httpretrier.RetryIfForWriteRequests),
		httpretrier.WithAttempts(c.retryAttempts),
	)
}

// sendRequest sends a request to the DevLake API.
func sendRequest[T requestData](ctx context.Context, c *Client, urlStr string, request *T) error {
	if err := c.valid.ValidateStructCtx(ctx, request); err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}

	r, err := httpPostRequest(ctx, urlStr, c.apiKey, request)
	if err != nil {
		return err
	}

	hr, err := c.newWriteHTTPRetrier()
	if err != nil {
		return fmt.Errorf("create retrier: %w", err)
	}

	resp, err := hr.Do(r) //nolint:bodyclose
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}

	defer logging.Close(ctx, resp.Body, "error closing response body")

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("devlake client error - Code: %v, Status: %v", resp.StatusCode, resp.Status)
	}

	return nil
}

// SendDeployment register a deployment with DevLake.
func (c *Client) SendDeployment(ctx context.Context, request *DeploymentRequest) error {
	urlStr := fmt.Sprintf(c.deploymentRegURLFormat, request.ConnectionID)
	return sendRequest[DeploymentRequest](ctx, c, urlStr, request)
}

// SendIncident register an incident with DevLake.
func (c *Client) SendIncident(ctx context.Context, request *IncidentRequest) error {
	urlStr := fmt.Sprintf(c.incidentRegURLFormat, request.ConnectionID)
	return sendRequest[IncidentRequest](ctx, c, urlStr, request)
}

// SendIncidentClose closes an incident with DevLake.
func (c *Client) SendIncidentClose(ctx context.Context, request *IncidentRequestClose) error {
	if err := c.valid.ValidateStructCtx(ctx, request); err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}

	urlStr := fmt.Sprintf(c.incidentCloseURLFormat, request.ConnectionID, request.IssueKey)

	return sendRequest[IncidentRequest](ctx, c, urlStr, nil)
}
