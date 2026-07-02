package slack

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
)

// Default configuration values.
const (
	defaultPingURL     = "https://status.slack.com/api/v2.0.0/current"
	defaultTimeout     = 1 * time.Second
	defaultPingTimeout = 1 * time.Second
	failStatus         = "active"
	failService        = "Apps/Integrations/APIs"
)

// HTTPClient is the minimal HTTP transport contract used by [Client].
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client sends Slack webhook messages and performs Slack status health checks.
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

// New constructs a Slack webhook client with defaults for timeout, retries, and optional message metadata.
// Parameters other than addr are optional defaults that can be overridden per Send call.
func New(addr, username, iconEmoji, iconURL, channel string, opts ...Option) (*Client, error) {
	address, err := url.ParseRequestURI(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse addr: %w", err)
	}

	// Reject relative or schemeless addresses at construction time; otherwise
	// every Send would fail at request time with an obscure transport error.
	// The addr is not echoed in the message because the webhook URL is a secret.
	if address.Scheme == "" || address.Host == "" {
		return nil, errors.New("invalid webhook addr: missing scheme or host")
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

// status models Slack status API response payload used by HealthCheck.
type status struct {
	Status string `json:"status"`

	// Services models a specific (and possibly outdated) shape of the Slack
	// status API response and may not match the current live API. The exported
	// type is preserved for backward compatibility; callers needing richer
	// incident data should verify the field against the current Slack status
	// API (https://status.slack.com) and decode the response independently.
	Services map[int]string `json:"services,omitempty"`
}

// HealthCheck verifies Slack status endpoint availability and checks for active API/app incidents.
//
// It models a specific (and possibly outdated) shape of the Slack status API
// response via the status type. Because the live Slack status API may have
// changed, callers needing richer or more reliable incident data should verify
// the response shape against the current Slack status API and implement their
// own check if necessary.
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
		// Drain any remaining body so the connection can be reused by keep-alive.
		_, _ = io.Copy(io.Discard, resp.Body)
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

// message is the outgoing webhook payload schema.
type message struct {
	Text      string `json:"text"`
	Username  string `json:"username,omitempty"`
	IconEmoji string `json:"icon_emoji,omitempty"`
	IconURL   string `json:"icon_url,omitempty"`
	Channel   string `json:"channel,omitempty"`
}

// Send posts a message to Slack webhook, using client defaults for empty metadata arguments.
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

// sendData serializes and posts the webhook payload with retry and status validation.
func (c *Client) sendData(ctx context.Context, reqData *message) (err error) {
	reqBody, _ := json.Marshal(reqData) //nolint:errchkjson

	r, nerr := http.NewRequestWithContext(ctx, http.MethodPost, c.address, bytes.NewReader(reqBody))
	if nerr != nil {
		// URL parse errors quote the full webhook address (a secret): redact it.
		return c.redactWebhookURL(fmt.Errorf("create request: %w", nerr))
	}

	r.Header.Set(httputil.HeaderContentType, httputil.MimeTypeJSON)

	hr, werr := c.newWriteHTTPRetrier()
	if werr != nil {
		return fmt.Errorf("create retrier: %w", werr)
	}

	resp, derr := hr.Do(r)
	if derr != nil {
		// Transport failures are wrapped by Go's HTTP client in *url.Error,
		// whose Error() includes the full request URL. The Slack webhook URL
		// path is a secret, so redact it before returning.
		return c.redactWebhookURL(fmt.Errorf("execute request: %w", derr))
	}

	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unable to send the message- Code: %v, Status: %v", resp.StatusCode, resp.Status)
	}

	return nil
}

// redactWebhookURL returns an error whose message has every occurrence of the
// client's webhook address replaced with "REDACTED". This guards against the
// secret leaking through wrapped transport errors: Go's HTTP client returns a
// *url.Error whose Error() string contains the full request URL, and the path
// of a Slack webhook URL is a secret. The original error is preserved via
// Unwrap so callers can still inspect it with errors.Is / errors.As.
func (c *Client) redactWebhookURL(err error) error {
	if err == nil {
		return nil
	}

	if !strings.Contains(err.Error(), c.address) {
		return err
	}

	return &redactedError{
		msg: strings.ReplaceAll(err.Error(), c.address, "REDACTED"),
		err: err,
	}
}

// redactedError wraps an error with a sanitized message while keeping the
// original error reachable through Unwrap for errors.Is / errors.As checks.
type redactedError struct {
	err error
	msg string
}

// Error returns the sanitized error message with the webhook URL redacted.
func (e *redactedError) Error() string { return e.msg }

// Unwrap exposes the original error for errors.Is / errors.As.
func (e *redactedError) Unwrap() error { return e.err }

// newWriteHTTPRetrier builds the write-oriented retrier for webhook delivery.
func (c *Client) newWriteHTTPRetrier() (*httpretrier.HTTPRetrier, error) {
	//nolint:wrapcheck
	return httpretrier.New(
		c.httpClient,
		httpretrier.WithRetryIfFn(httpretrier.RetryIfForWriteRequests),
		httpretrier.WithAttempts(c.retryAttempts),
	)
}
