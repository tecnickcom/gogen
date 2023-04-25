// Package sleuth contains the implementation of a basic Sleuth.io API client.
// Ref.: https://help.sleuth.io/sleuth-api
package sleuth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Vonage/gosrvlib/pkg/logging"
)

// DeployRegistrationRequest defines the request for Deploy Registration.
type DeployRegistrationRequest struct {
	// Deployment is the Sleuth deploymnet ID as found in the Sleuth URL, following the prefix https://app.sleuth.io/org_slug/deployments/.
	Deployment string `json:"-" validate:"required,max=50"`

	// Sha is the Git SHA of the commit to be registered as a deploy.
	Sha string `json:"sha" validate:"required,max=40"`

	// Environment is the environment to register the deploy against.
	// If not provided Sleuth will use the default environment of the Project.
	Environment string `json:"environment,omitempty" validate:"omitempty,max=50"`

	// Date is the ISO 8601 deployment date and time string.
	Date string `json:"date,omitempty" validate:"omitempty,datetime=2006-01-02 15:04:05"`

	// Tags is a comma-delimited list of tags.
	// Default to tags calculated by matching paths defined in the .sleuth/TAGS file.
	Tags []string `json:"tags,omitempty" validate:"omitempty,max=50,dive,max=50,startswith=#"`

	// IgnoreIfDuplicate ignores duplicate SHA and do not return an error.
	IgnoreIfDuplicate bool `json:"ignore_if_duplicate,omitempty" validate:"omitempty"`

	// Email is the email address of the author.
	Email string `json:"email,omitempty" validate:"omitempty,email"`

	// Links contains key/value pair consisting of the link name and the link itself.
	Links map[string]string `json:"links,omitempty" validate:"omitempty,max=50,dive,url"`
}

// ManualChangeRequest defines the request for Manual Change.
type ManualChangeRequest struct {
	// Project is the Sleuth project ID as found in the Sleuth URL, following the prefix https://app.sleuth.io/org_slug/.
	Project string `json:"-" validate:"required,max=50"`

	// Name is the title for the manual change.
	Name string `json:"name" validate:"required,max=255"`

	// Description for manual changes. Omit if using SHA instead.
	Description string `json:"description,omitempty" validate:"omitempty,max=65535"`

	// Environment is the environment to register the deploy against.
	// If not provided Sleuth will use the default environment of the Project.
	Environment string `json:"environment,omitempty" validate:"omitempty,max=50"`

	// Tags is a comma-delimited list of tags.
	// Default to tags calculated by matching paths defined in the .sleuth/TAGS file.
	Tags []string `json:"tags,omitempty" validate:"omitempty,max=50,dive,max=50,startswith=#"`

	// Author is the email address of the change author.
	Author string `json:",omitempty" validate:"omitempty,email"`

	// Email is the email address of the user associated with the project receiving the manual change.
	Email string `json:"email,omitempty" validate:"omitempty,email"`
}

// IncidentType defines the valid values for an Incident Type.
type IncidentType string

const (
	// Triggered indicates a new incident.
	Triggered IncidentType = "triggered"

	// Resolved indicates the incident has been resolved.
	Resolved IncidentType = "resolved"

	// Reopened indicated the incident has been reopened.
	Reopened IncidentType = "reopened"
)

// CustomIncidentImpactRegistrationRequest defines the request for Custom Incident Impact Registration.
type CustomIncidentImpactRegistrationRequest struct {
	// Project is the Sleuth project ID as found in the Sleuth URL, following the prefix https://app.sleuth.io/org_slug/.
	Project string `json:"-" validate:"required,max=50"`

	// Environment is the environment to register the deploy against.
	// Found at the end of the URL of the Sleuth org when navigating to the target project
	// and selecting the target custom incident impact source: env_slug=ENVIRONMENT_SLUG.
	Environment string `json:"-" validate:"required,max=50"`

	// ImpactSource is found in the URL of the Sleuth org when navigating to the target project
	// and selecting the target custom incident impact source, just before the ?env_slug.
	ImpactSource string `json:"-" validate:"required,max=50"`

	// Type Valid types are triggered, resolved, and reopened.
	Type IncidentType `json:"type" validate:"required,oneof=triggered resolved reopened"`

	// ID is the unique (custom) incident identifier.
	ID string `json:"id" validate:"omitempty,max=50"`

	// Date is the ISO 8601 date and time string when the event occurred.
	// Defaults to the current time.
	Date string `json:"date,omitempty" validate:"omitempty,datetime=2006-01-02 15:04:05"`

	// EndedDate is the ISO 8601 date and time string when the event ended.
	// Use it with "type": "triggered" to register past incident event.
	EndedDate string `json:"ended_date,omitempty" validate:"omitempty,datetime=2006-01-02 15:04:05"`

	// Title is the human-readable title of the incident.
	Title string `json:"title,omitempty" validate:"omitempty,max=255"`

	// URL to the incident in the external system.
	URL string `json:"url,omitempty" validate:"omitempty,url"`
}

// CustomMetricImpactRegistrationRequest defines the request for Custom Metric Impact Registration.
type CustomMetricImpactRegistrationRequest struct {
	// ImpactID is the integer ID that can be found bny navigating in the Custom Metric Impact Source,
	// clicking the gearwheel icon in the top-right corner, and selecting "Show register details".
	ImpactID int `json:"-" validate:"required"`

	// Value is the metric value to be registered.
	Value float64 `json:"value" validate:"required"`

	// Date is the ISO 8601 date and time string at which the metric value should be registered.
	// Defaults to the current time.
	Date string `json:"date,omitempty" validate:"omitempty,datetime=2006-01-02 15:04:05"`
}

// requestData aggregates the types for different API requests.
type requestData interface {
	DeployRegistrationRequest | ManualChangeRequest | CustomIncidentImpactRegistrationRequest | CustomMetricImpactRegistrationRequest
}

// httpRequest prepare an HTTP request encoding the payload as JSON.
func httpRequest(ctx context.Context, urlStr, apiKey string, request any) (*http.Request, error) {
	buffer := &bytes.Buffer{}

	if err := json.NewEncoder(buffer).Encode(request); err != nil {
		return nil, fmt.Errorf("json encoding: %w", err)
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, buffer)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	r.Header.Set(headerAuthorization, apiKey)
	r.Header.Set(headerContentType, contentType)

	return r, nil
}

// sendRequest sends a request to the Sleuth API.
func sendRequest[T requestData](ctx context.Context, c *Client, urlStr string, request *T) error {
	if err := c.valid.ValidateStruct(request); err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}

	r, err := httpRequest(ctx, urlStr, c.apiKey, request)
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
