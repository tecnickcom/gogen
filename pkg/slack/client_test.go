package slack

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Vonage/gosrvlib/pkg/httputil"
	"github.com/Vonage/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
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
		tt := tt
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
			pingHandlerDelay:      timeout + 1,
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
			name:                  "fails because bad status for API service",
			pingHandlerStatusCode: http.StatusOK,
			pingBody:              &status{Status: failStatus, Services: map[int]string{0: failService}},
			wantErr:               true,
		},
		{
			name:                  "fails because bad status for multiple services",
			pingHandlerStatusCode: http.StatusOK,
			pingBody:              &status{Status: failStatus, Services: map[int]string{0: "Calls", 1: failService, 2: "Search"}},
			wantErr:               true,
		},
		{
			name:                  "success with bad status on another service",
			pingHandlerStatusCode: http.StatusOK,
			pingBody:              &status{Status: failStatus, Services: map[int]string{0: "Calls"}},
			wantErr:               false,
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					httputil.SendStatus(r.Context(), w, http.StatusMethodNotAllowed)
					return
				}

				if tt.pingHandlerDelay != 0 {
					time.Sleep(tt.pingHandlerDelay)
				}

				httputil.SendJSON(r.Context(), w, tt.pingHandlerStatusCode, tt.pingBody)
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

			err = c.HealthCheck(testutil.Context())
			if tt.wantErr {
				require.Error(t, err, "Client.HealthCheck() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				require.NoError(t, err, "Client.HealthCheck() unexpected error = %v", err)
			}
		})
	}
}

//nolint:contextcheck
func TestClient_Send(t *testing.T) {
	t.Parallel()

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
			webhookHandler: func(w http.ResponseWriter, r *http.Request) {
				httputil.SendStatus(testutil.Context(), w, http.StatusInternalServerError)
			},
			text:      "text 1",
			username:  "",
			iconEmoji: "",
			iconURL:   "",
			channel:   "",
			wantErr:   true,
		},
		{
			name: "fails because of timeout",
			webhookHandler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(timeout + 1)
				httputil.SendStatus(testutil.Context(), w, http.StatusOK)
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
			webhookHandler: func(w http.ResponseWriter, r *http.Request) {
				httputil.SendStatus(testutil.Context(), w, http.StatusOK)
			},
			text:       "text address",
			clientFunc: func(c *Client) *Client { c.address = "*&^%-ERROR-"; return c },
			wantErr:    true,
		},
		{
			name: "fails because WriteHTTPRetrier error",
			webhookHandler: func(w http.ResponseWriter, r *http.Request) {
				httputil.SendStatus(testutil.Context(), w, http.StatusOK)
			},
			text:       "text retrier",
			clientFunc: func(c *Client) *Client { c.retryAttempts = 0; return c },
			wantErr:    true,
		},
		{
			name: "succeed with valid response",
			webhookHandler: func(w http.ResponseWriter, r *http.Request) {
				httputil.SendStatus(testutil.Context(), w, http.StatusOK)
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
		tt := tt
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

			err = c.Send(testutil.Context(), tt.text, tt.username, tt.iconEmoji, tt.iconURL, tt.channel)
			if tt.wantErr {
				require.Error(t, err, "error = %v, wantErr %v", err, tt.wantErr)
			} else {
				require.NoError(t, err, "unexpected error = %v", err)
			}
		})
	}
}
