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
	"regexp"
	"time"

	"github.com/Vonage/gosrvlib/pkg/httpretrier"
	"github.com/Vonage/gosrvlib/pkg/logging"
	"github.com/Vonage/gosrvlib/pkg/validator"
)

const (
	defaultTimeout          = 1 * time.Minute
	defaultPingTimeout      = 15 * time.Second
	headerAuthorization     = "Authorization"
	headerContentType       = "Content-Type"
	headerAccept            = "Accept"
	contentType             = "application/json"
	regexPatternHealthcheck = "Deployment - Not Found"
)

// HTTPClient contains the function to perform the actual HTTP request.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client represents the config options required by this client.
type Client struct {
	httpClient                  HTTPClient
	baseURL                     *url.URL
	regexHealthcheck            *regexp.Regexp
	valid                       *validator.Validator
	timeout                     time.Duration
	pingTimeout                 time.Duration
	retryDelay                  time.Duration
	retryAttempts               uint
	apiKey                      string
	pingURL                     string
	deployRegistrationURLFormat string
	manualChangeURLFormat       string
	customIncidentURLFormat     string
	customMetricURLFormat       string
}

// New creates a new client instance.
// Example for addr: "https://app.sleuth.io/api/1"
func New(addr, org, apiKey string, opts ...Option) (*Client, error) {
	baseURL, err := url.Parse(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse addr: %w", err)
	}

	if org == "" {
		return nil, errors.New("org is empty")
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
		regexHealthcheck:            regexp.MustCompile(regexPatternHealthcheck),
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

// HealthCheck performs a status check on this service.
// Note: sleuth.io API currently does not provide a ping endpoint,
// so we check if we are getting the right error using the
// correct API Key and inexistent deployment ID.
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.pingTimeout)
	defer cancel()

	req, err := httpPostRequest(
		ctx,
		c.pingURL,
		c.apiKey,
		&DeployRegistrationRequest{
			Sha:               "0",
			Environment:       "TEST",
			IgnoreIfDuplicate: true,
		},
	)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req) //nolint:bodyclose
	if err != nil {
		return fmt.Errorf("healthcheck request: %w", err)
	}

	defer logging.Close(ctx, resp.Body, "error while closing HealthCheck response body")

	if resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("unexpected healthcheck status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed reading response body: %w", err)
	}

	if !c.regexHealthcheck.MatchString(string(body)) {
		return fmt.Errorf("unexpected healthcheck response: %v", string(body))
	}

	return nil
}

func (c *Client) newWriteHTTPRetrier() (*httpretrier.HTTPRetrier, error) {
	//nolint:wrapcheck
	return httpretrier.New(
		c.httpClient,
		httpretrier.WithRetryIfFn(httpretrier.RetryIfForWriteRequests),
		httpretrier.WithAttempts(c.retryAttempts),
	)
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

	r.Header.Set(headerAuthorization, "apikey "+apiKey)
	r.Header.Set(headerContentType, contentType)
	r.Header.Set(headerAccept, contentType)

	return r, nil
}

// sendRequest sends a request to the Sleuth API.
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
		return fmt.Errorf("sleuth client error - Code: %v, Status: %v", resp.StatusCode, resp.Status)
	}

	return nil
}

// SendDeployRegistration register a deployment with Sleuth.
func (c *Client) SendDeployRegistration(ctx context.Context, request *DeployRegistrationRequest) error {
	urlStr := fmt.Sprintf(c.deployRegistrationURLFormat, request.Deployment)
	return sendRequest[DeployRegistrationRequest](ctx, c, urlStr, request)
}

// SendManualChange register a manual change with Sleuth.
// Manual changes are any changes not tracked by source code, feature flags,
// or any other type of change not supported by Sleuth.
func (c *Client) SendManualChange(ctx context.Context, request *ManualChangeRequest) error {
	urlStr := fmt.Sprintf(c.manualChangeURLFormat, request.Project)
	return sendRequest[ManualChangeRequest](ctx, c, urlStr, request)
}

// SendCustomIncidentImpactRegistration register Custom Incident Impact values.
// Allows to submit custom incident impact values to Sleuth to get Failure Rate and MTTR values.
func (c *Client) SendCustomIncidentImpactRegistration(ctx context.Context, request *CustomIncidentImpactRegistrationRequest) error {
	urlStr := fmt.Sprintf(c.customIncidentURLFormat, request.Project, request.Environment, request.ImpactSource, c.apiKey)
	return sendRequest[CustomIncidentImpactRegistrationRequest](ctx, c, urlStr, request)
}

// SendCustomMetricImpactRegistration register Custom Metric Impact values.
// Allows to submit custom metric impact values to Sleuth.
// Sleuth will perform anomaly detection on these values and they will inform the health of the deployments.
func (c *Client) SendCustomMetricImpactRegistration(ctx context.Context, request *CustomMetricImpactRegistrationRequest) error {
	urlStr := fmt.Sprintf(c.customMetricURLFormat, request.ImpactID)
	return sendRequest[CustomMetricImpactRegistrationRequest](ctx, c, urlStr, request)
}
