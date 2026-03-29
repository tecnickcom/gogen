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

	"github.com/tecnickcom/gogen/pkg/httpretrier"
	"github.com/tecnickcom/gogen/pkg/httputil"
	"github.com/tecnickcom/gogen/pkg/validator"
)

// Constants for default timeouts and regex patterns.
const (
	defaultTimeout          = 1 * time.Minute
	defaultPingTimeout      = 15 * time.Second
	regexPatternHealthcheck = "Deployment - Not Found"
)

// HTTPClient is the minimal HTTP transport contract used by [Client].
type HTTPClient interface {
	// Do sends an HTTP request and returns an HTTP response.
	Do(req *http.Request) (*http.Response, error)
}

// Client sends deployment/change/impact events to the Sleuth API.
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

// New constructs a Sleuth API client with validation, retry defaults, and URL templates for the provided org.
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
		return fmt.Errorf("healthcheck request: %w", derr)
	}

	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()

	if resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("unexpected healthcheck status code: %d", resp.StatusCode)
	}

	body, rerr := io.ReadAll(resp.Body)
	if rerr != nil {
		return fmt.Errorf("failed reading response body: %w", rerr)
	}

	if !c.regexHealthcheck.MatchString(string(body)) {
		return fmt.Errorf("unexpected healthcheck response: %v", string(body))
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

	httputil.AddJsonHeaders(r)
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
		return fmt.Errorf("execute request: %w", derr)
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

// newWriteHTTPRetrier creates a write-oriented HTTP retrier using configured attempt count.
func (c *Client) newWriteHTTPRetrier() (*httpretrier.HTTPRetrier, error) {
	//nolint:wrapcheck
	return httpretrier.New(
		c.httpClient,
		httpretrier.WithRetryIfFn(httpretrier.RetryIfForWriteRequests),
		httpretrier.WithAttempts(c.retryAttempts),
	)
}
