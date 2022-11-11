// Package slack is a basic Slack API client to send messages via a Webhook.
package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/nexmoinc/gosrvlib/pkg/httpretrier"
	"github.com/nexmoinc/gosrvlib/pkg/logging"
)

const (
	contentType        = "Content-Type"
	mimeTypeJSON       = "application/json"
	defaultPingURL     = "https://status.slack.com/api/v2.0.0/current"
	defaultTimeout     = 1 * time.Second
	defaultPingTimeout = 1 * time.Second
	failStatus         = "active"
	failService        = "Apps/Integrations/APIs"
)

// HTTPClient contains the function to perform the actual HTTP request.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Client is the implementation of the service client.
type Client struct {
	httpClient    HTTPClient
	address       string
	timeout       time.Duration
	pingTimeout   time.Duration
	retryAttempts uint
	pingURL       string
	username      string
	iconEmoji     string
	iconURL       string
	channel       string
}

// New creates a new instance of the Slack service client.
// The arguments other than "addr" are optional. They can be set in the Webhook configuration or in each individual message.
func New(addr, username, iconEmoji, iconURL, channel string, opts ...Option) (*Client, error) {
	address, err := url.Parse(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse addr: %w", err)
	}

	c := &Client{
		address:       address.String(),
		timeout:       defaultTimeout,
		pingTimeout:   defaultPingTimeout,
		retryAttempts: httpretrier.DefaultAttempts,
		pingURL:       defaultPingURL,
		username:      username,
		iconEmoji:     iconEmoji,
		iconURL:       iconURL,
		channel:       channel,
	}

	for _, applyOpt := range opts {
		applyOpt(c)
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: c.timeout}
	}

	return c, nil
}

type status struct {
	Status   string         `json:"status"`
	Services map[int]string `json:"services,omitempty"`
}

// HealthCheck performs a status check on the Slack service.
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.pingTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.pingURL, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	resp, err := c.httpClient.Do(req) //nolint:bodyclose
	if err != nil {
		return fmt.Errorf("healthcheck request: %w", err)
	}

	defer logging.Close(ctx, resp.Body, "error while closing HealthCheck response body")

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected healthcheck status code: %d", resp.StatusCode)
	}

	respBody := &status{}

	if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
		return fmt.Errorf("failed decoding response body: %w", err)
	}

	if respBody.Status == failStatus {
		for _, service := range respBody.Services {
			if service == failService {
				return fmt.Errorf("unexpected healthcheck status: %v", respBody.Status)
			}
		}
	}

	return nil
}

// Message contains the message payload.
type message struct {
	Text      string `json:"text"`
	Username  string `json:"username,omitempty"`
	IconEmoji string `json:"icon_emoji,omitempty"`
	IconURL   string `json:"icon_url,omitempty"`
	Channel   string `json:"channel,omitempty"`
}

// Send a message accounting for the default values.
// The arguments after "text" can be left empty to get the default values.
func (c *Client) Send(ctx context.Context, text, username, iconEmoji, iconURL, channel string) error {
	reqData := &message{
		Text:      text,
		Username:  stringValueOrDefault(username, c.username),
		IconEmoji: stringValueOrDefault(iconEmoji, c.iconEmoji),
		IconURL:   stringValueOrDefault(iconURL, c.iconURL),
		Channel:   stringValueOrDefault(channel, c.channel),
	}

	return c.sendData(ctx, reqData)
}

// sendData sends the specified data.
func (c *Client) sendData(ctx context.Context, reqData *message) error {
	reqBody, _ := json.Marshal(reqData) //nolint:errchkjson

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, c.address, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	r.Header.Set(contentType, mimeTypeJSON)

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
		return fmt.Errorf("unable to send the message- Code: %v, Status: %v", resp.StatusCode, resp.Status)
	}

	return nil
}

func (c *Client) newWriteHTTPRetrier() (*httpretrier.HTTPRetrier, error) {
	return httpretrier.New(c.httpClient, httpretrier.WithRetryIfFn(httpretrier.RetryIfForWriteRequests), httpretrier.WithAttempts(c.retryAttempts)) //nolint:wrapcheck
}

func stringValueOrDefault(v, def string) string {
	if v == "" {
		return def
	}

	return v
}
