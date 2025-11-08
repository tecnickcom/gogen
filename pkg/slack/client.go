package slack

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
)

// Default configuration values.
const (
	defaultPingURL     = "https://status.slack.com/api/v2.0.0/current"
	defaultTimeout     = 1 * time.Second
	defaultPingTimeout = 1 * time.Second
	failStatus         = "active"
	failService        = "Apps/Integrations/APIs"
)

// HTTPClient contains the function to perform the actual HTTP request.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
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
// The arguments other than "addr" (Slack Webhook URL) are optional,
// they can be set in the Webhook configuration or in each individual message.
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

// status represents the response from the health check endpoint.
type status struct {
	Status   string         `json:"status"`
	Services map[int]string `json:"services,omitempty"`
}

// HealthCheck performs a status check on this service.
func (c *Client) HealthCheck(ctx context.Context) (err error) {
	ctx, cancel := context.WithTimeout(ctx, c.pingTimeout)
	defer cancel()

	req, nerr := http.NewRequestWithContext(ctx, http.MethodGet, c.pingURL, nil)
	if nerr != nil {
		return fmt.Errorf("build request: %w", nerr)
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

	respBody := &status{}

	rerr := json.NewDecoder(resp.Body).Decode(respBody)
	if rerr != nil {
		return fmt.Errorf("failed decoding response body: %w", rerr)
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
		Username:  httputil.StringValueOrDefault(username, c.username),
		IconEmoji: httputil.StringValueOrDefault(iconEmoji, c.iconEmoji),
		IconURL:   httputil.StringValueOrDefault(iconURL, c.iconURL),
		Channel:   httputil.StringValueOrDefault(channel, c.channel),
	}

	return c.sendData(ctx, reqData)
}

// sendData sends the specified data.
func (c *Client) sendData(ctx context.Context, reqData *message) (err error) {
	reqBody, _ := json.Marshal(reqData) //nolint:errchkjson

	r, nerr := http.NewRequestWithContext(ctx, http.MethodPost, c.address, bytes.NewReader(reqBody))
	if nerr != nil {
		return fmt.Errorf("create request: %w", nerr)
	}

	r.Header.Set(httputil.HeaderContentType, httputil.MimeTypeJSON)

	hr, werr := c.newWriteHTTPRetrier()
	if werr != nil {
		return fmt.Errorf("create retrier: %w", werr)
	}

	resp, derr := hr.Do(r)
	if derr != nil {
		return fmt.Errorf("execute request: %w", derr)
	}

	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unable to send the message- Code: %v, Status: %v", resp.StatusCode, resp.Status)
	}

	return nil
}

// newWriteHTTPRetrier creates a new HTTP retrier for write requests.
func (c *Client) newWriteHTTPRetrier() (*httpretrier.HTTPRetrier, error) {
	//nolint:wrapcheck
	return httpretrier.New(
		c.httpClient,
		httpretrier.WithRetryIfFn(httpretrier.RetryIfForWriteRequests),
		httpretrier.WithAttempts(c.retryAttempts),
	)
}
