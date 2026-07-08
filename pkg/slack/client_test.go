package slack

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/gogen/pkg/httputil"
	"github.com/tecnickcom/gogen/pkg/testutil"
)

func TestNew(t *testing.T) {
	t.Parallel()

	timeout := 100 * time.Millisecond

	tests := []struct {
		name        string
		serviceAddr string
		opts        []Option
		wantTimeout time.Duration
		wantErr     bool
	}{
		{
			name:        "fails with invalid character in URL",
			serviceAddr: "http://invalid-url.domain.invalid\u007F",
			wantErr:     true,
		},
		{
			name:        "fails with empty addr",
			serviceAddr: "",
			wantErr:     true,
		},
		{
			name:        "fails with relative addr missing scheme and host",
			serviceAddr: "/services/T0000/B0000/token",
			wantErr:     true,
		},
		{
			name:        "fails with scheme but missing host",
			serviceAddr: "http:///services/T0000/B0000/token",
			wantErr:     true,
		},
		{
			name:        "succeeds with defaults",
			serviceAddr: "http://service.domain.invalid:1234",
			wantTimeout: defaultTimeout,
			wantErr:     false,
		},
		{
			name:        "succeeds with overridden timeouts",
			serviceAddr: "http://service.domain.invalid:1234",
			opts:        []Option{WithTimeout(timeout), WithPingTimeout(timeout)},
			wantTimeout: timeout,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.opts = append(tt.opts, WithRetryAttempts(1))
			c, err := New(
				tt.serviceAddr,
				"default-username",
				":default-iconEmoji:",
				"https://default.iconURL.invalid",
				"#default-channel",
				tt.opts...,
			)

			if tt.wantErr {
				require.Nil(t, c, "New() returned client should be nil")
				require.Error(t, err, "New() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			require.NotNil(t, c, "New() returned client should not be nil")
			require.NoError(t, err, "New() unexpected error = %v", err)
		})
	}
}

func TestNewInvalidRetryConfig(t *testing.T) {
	t.Parallel()

	// The retrier is built during New, so an invalid retry setting (0 attempts)
	// is rejected at construction instead of failing on every Send.
	c, err := New(
		"http://service.domain.invalid:1234",
		"default-username",
		":default-iconEmoji:",
		"https://default.iconURL.invalid",
		"#default-channel",
		WithRetryAttempts(0),
	)

	require.Nil(t, c, "New() returned client should be nil")
	require.Error(t, err, "New() should fail with invalid retry attempts")
}

//nolint:gocognit
func TestClient_HealthCheck(t *testing.T) {
	t.Parallel()

	timeout := 100 * time.Millisecond

	tests := []struct {
		name                  string
		pingHandlerDelay      time.Duration
		pingHandlerStatusCode int
		pingURL               string
		pingBody              any
		wantErr               bool
	}{
		{
			name:                  "returns error because of timeout",
			pingHandlerDelay:      timeout + 50*time.Millisecond, // margin absorbs timer jitter under load
			pingHandlerStatusCode: http.StatusOK,
			pingBody:              &status{Status: "ok"},
			wantErr:               true,
		},
		{
			name:                  "returns error from endpoint",
			pingHandlerStatusCode: http.StatusInternalServerError,
			pingBody:              &status{Status: "ok"},
			wantErr:               true,
		},
		{
			name:                  "fails because ping url error",
			pingHandlerStatusCode: http.StatusOK,
			pingURL:               "%^*&-ERROR",
			pingBody:              &status{Status: "ok"},
			wantErr:               true,
		},
		{
			name:                  "fails because active incident affects API service",
			pingHandlerStatusCode: http.StatusOK,
			pingBody: &status{Status: "active", ActiveIncidents: []incident{
				{Status: "active", Title: "API degraded", URL: "https://slack-status.com/x", Services: []string{failService}},
			}},
			wantErr: true,
		},
		{
			name:                  "fails because active incident lists API among multiple services",
			pingHandlerStatusCode: http.StatusOK,
			pingBody: &status{Status: "active", ActiveIncidents: []incident{
				{Status: "active", Services: []string{"Calls", failService, "Search"}},
			}},
			wantErr: true,
		},
		{
			name:                  "fails because incident with unset status affects API service",
			pingHandlerStatusCode: http.StatusOK,
			pingBody: &status{Status: "active", ActiveIncidents: []incident{
				{Services: []string{failService}},
			}},
			wantErr: true,
		},
		{
			name:                  "success when active incident affects another service only",
			pingHandlerStatusCode: http.StatusOK,
			pingBody: &status{Status: "active", ActiveIncidents: []incident{
				{Status: "active", Services: []string{"Calls"}},
			}},
			wantErr: false,
		},
		{
			name:                  "success when API incident is already resolved",
			pingHandlerStatusCode: http.StatusOK,
			pingBody: &status{Status: "active", ActiveIncidents: []incident{
				{Status: "resolved", Services: []string{failService}},
			}},
			wantErr: false,
		},
		{
			name:                  "fails because bad response body",
			pingHandlerStatusCode: http.StatusOK,
			pingBody:              "{",
			wantErr:               true,
		},
		{
			name:                  "returns success from endpoint",
			pingHandlerStatusCode: http.StatusOK,
			pingBody:              &status{Status: "ok"},
			wantErr:               false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hres := httputil.NewHTTPResp(slog.Default())
			mux := http.NewServeMux()

			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					hres.SendStatus(r.Context(), w, http.StatusMethodNotAllowed)
					return
				}

				if tt.pingHandlerDelay != 0 {
					time.Sleep(tt.pingHandlerDelay)
				}

				hres.SendJSON(r.Context(), w, tt.pingHandlerStatusCode, tt.pingBody)
			})

			ts := httptest.NewServer(mux)
			defer ts.Close()

			c, err := New(
				ts.URL,
				"",
				"",
				"",
				"",
				WithRetryAttempts(1),
				WithPingURL(ts.URL),
				WithTimeout(timeout),
				WithPingTimeout(timeout),
			)
			require.NoError(t, err, "Client.HealthCheck() create client unexpected error = %v", err)

			if tt.pingURL != "" {
				c.pingURL = tt.pingURL
			}

			err = c.HealthCheck(t.Context())
			if tt.wantErr {
				require.Error(t, err, "Client.HealthCheck() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				require.NoError(t, err, "Client.HealthCheck() unexpected error = %v", err)
			}
		})
	}
}

func TestClient_redactWebhookURL(t *testing.T) {
	t.Parallel()

	webhookURL := "http://hooks.test.invalid/services/T0000/B0000/secret-webhook-token"

	c, err := New(
		webhookURL,
		"",
		"",
		"",
		"",
		WithRetryAttempts(1),
	)
	require.NoError(t, err)

	require.NoError(t, c.redactWebhookURL(nil), "redactWebhookURL(nil) should return nil")

	// Errors that do not contain the webhook URL must be returned unchanged.
	plain := errors.New("some transport failure")
	require.ErrorIs(t, c.redactWebhookURL(plain), plain, "errors without the webhook URL must be passed through unchanged")

	// Errors that contain the webhook URL must have it redacted, while
	// remaining unwrappable to the original error.
	secret := fmt.Errorf("execute request: Post %q: connection refused", webhookURL)
	got := c.redactWebhookURL(secret)
	require.Error(t, got)
	require.NotContains(t, got.Error(), webhookURL, "redacted error must not contain the webhook URL")
	require.NotContains(t, got.Error(), "secret-webhook-token", "redacted error must not contain the webhook path")
	require.Contains(t, got.Error(), "REDACTED", "redacted error must mention REDACTED")
	require.ErrorIs(t, got, secret, "redacted error must unwrap to the original error")
}

// TestClient_Send_redactsWebhookURL forces a real transport error (connection
// refused) whose *url.Error message embeds the full webhook URL, and asserts
// the error surfaced by Send does not leak the secret webhook path.
func TestClient_Send_redactsWebhookURL(t *testing.T) {
	t.Parallel()

	// Start and immediately close a test server so the webhook address points
	// to a closed port, forcing a transport error embedding the request URL.
	ts := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	webhookURL := ts.URL + "/services/T0000/B0000/secret-webhook-token"

	ts.Close()

	c, err := New(
		webhookURL,
		"",
		"",
		"",
		"",
		WithRetryAttempts(1),
		WithTimeout(100*time.Millisecond),
	)
	require.NoError(t, err)

	err = c.Send(t.Context(), "test message", "", "", "", "")

	require.Error(t, err)
	require.NotContains(t, err.Error(), "secret-webhook-token", "Send error must not leak the webhook path")
	require.NotContains(t, err.Error(), webhookURL, "Send error must not leak the webhook URL")
	require.Contains(t, err.Error(), "REDACTED", "Send error must redact the webhook URL")
}

//nolint:contextcheck
func TestClient_Send(t *testing.T) {
	t.Parallel()

	hres := httputil.NewHTTPResp(slog.Default())

	timeout := 100 * time.Millisecond

	tests := []struct {
		name           string
		webhookHandler http.HandlerFunc
		text           string
		username       string
		iconEmoji      string
		iconURL        string
		channel        string
		clientFunc     func(c *Client) *Client
		wantErr        bool
	}{
		{
			name: "fails because status not OK",
			webhookHandler: func(w http.ResponseWriter, _ *http.Request) {
				hres.SendStatus(t.Context(), w, http.StatusInternalServerError)
			},
			text:      "text 1",
			username:  "",
			iconEmoji: "",
			iconURL:   "",
			channel:   "",
			wantErr:   true,
		},
		{
			name: "fails because status not OK with empty body",
			webhookHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			text:    "text empty body",
			wantErr: true,
		},
		{
			name: "fails because of timeout",
			webhookHandler: func(w http.ResponseWriter, _ *http.Request) {
				time.Sleep(timeout + 50*time.Millisecond) // margin absorbs timer jitter under load
				hres.SendStatus(t.Context(), w, http.StatusOK)
			},
			text:      "text TIMEOUT",
			username:  "timeout-username",
			iconEmoji: ":timeout-iconEmoji:",
			iconURL:   "https://timeout.iconURL.invalid",
			channel:   "#timeout-channel",
			wantErr:   true,
		},
		{
			name: "fails because bad address",
			webhookHandler: func(w http.ResponseWriter, _ *http.Request) {
				hres.SendStatus(t.Context(), w, http.StatusOK)
			},
			text:       "text address",
			clientFunc: func(c *Client) *Client { c.address = "*&^%-ERROR-"; return c },
			wantErr:    true,
		},
		{
			name: "succeed with valid response",
			webhookHandler: func(w http.ResponseWriter, _ *http.Request) {
				hres.SendStatus(t.Context(), w, http.StatusOK)
			},
			text:      "text OK",
			username:  "ok-username",
			iconEmoji: ":ok-iconEmoji:",
			iconURL:   "https://ok.iconURL.invalid",
			channel:   "#ok-channel",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mux := testutil.RouterWithHandler(http.MethodPost, "/", tt.webhookHandler)

			ts := httptest.NewServer(mux)
			defer ts.Close()

			c, err := New(
				ts.URL,
				"default-username",
				":default-iconEmoji:",
				"https://default.iconURL.invalid",
				"#default-channel",
				WithRetryAttempts(1),
				WithTimeout(timeout),
				WithPingTimeout(timeout),
			)
			require.NoError(t, err, "create client unexpected error = %v", err)

			if tt.clientFunc != nil {
				c = tt.clientFunc(c)
			}

			err = c.Send(t.Context(), tt.text, tt.username, tt.iconEmoji, tt.iconURL, tt.channel)
			if tt.wantErr {
				require.Error(t, err, "error = %v, wantErr %v", err, tt.wantErr)
			} else {
				require.NoError(t, err, "unexpected error = %v", err)
			}
		})
	}
}

// TestNew_sentinelErrors verifies New wraps its failures in the exported
// sentinels so callers can match them with errors.Is.
func TestNew_sentinelErrors(t *testing.T) {
	t.Parallel()

	_, err := New("http://", "", "", "", "", WithRetryAttempts(1))
	require.ErrorIs(t, err, ErrInvalidAddress)

	_, err = New("http://service.domain.invalid:1234", "", "", "", "", WithRetryAttempts(0))
	require.ErrorIs(t, err, ErrInvalidRetryConfig)
}

// TestNew_doesNotLeakWebhookURLOnParseError verifies that when the webhook
// address fails to parse, the secret path is not echoed in the returned error.
// The invalid percent-escape "%zz" triggers a url.Error whose Error() would
// normally embed the full webhook URL.
func TestNew_doesNotLeakWebhookURLOnParseError(t *testing.T) {
	t.Parallel()

	const secret = "secret-token-do-not-leak"

	_, err := New("http://hooks.test.invalid/services/T/B/"+secret+"%zz", "", "", "", "", WithRetryAttempts(1))

	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidAddress)
	require.NotContains(t, err.Error(), secret, "parse error must not leak the webhook path")
}
