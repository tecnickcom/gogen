package sleuth

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

	tests := []struct {
		name        string
		addr        string
		org         string
		apikey      string
		opts        []Option
		wantTimeout time.Duration
		wantErr     bool
	}{
		{
			name:    "fails with invalid character in URL",
			addr:    "http://invalid-url.domain.invalid\u007F",
			org:     "testorg",
			apikey:  "0123456789abcdef",
			wantErr: true,
		},
		{
			name:    "fails with empty org",
			addr:    "http://service.domain.invalid:1234",
			org:     "",
			apikey:  "0123456789abcdef",
			wantErr: true,
		},
		{
			name:    "fails with empty api key",
			addr:    "http://service.domain.invalid:1234",
			org:     "testorg",
			apikey:  "",
			wantErr: true,
		},
		{
			name:        "succeeds with defaults",
			addr:        "http://service.domain.invalid:1234",
			org:         "testorg",
			apikey:      "0123456789abcdef",
			wantTimeout: defaultPingTimeout,
			wantErr:     false,
		},
		{
			name:        "succeeds with options",
			addr:        "http://service.domain.invalid:1234",
			org:         "testorg",
			apikey:      "0123456789abcdef",
			opts:        []Option{WithPingTimeout(2 * time.Second)},
			wantTimeout: 2 * time.Second,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.opts = append(tt.opts, WithRetryAttempts(1))

			c, err := New(
				tt.addr,
				tt.org,
				tt.apikey,
				tt.opts...,
			)

			if tt.wantErr {
				require.Nil(t, c, "New() returned client should be nil")
				require.Error(t, err, "New() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			require.NotNil(t, c, "New() returned client should not be nil")
			require.NoError(t, err, "New() unexpected error = %v", err)
			require.Equal(t, tt.wantTimeout, c.pingTimeout, "New() unexpected pingTimeout = %d got %d", tt.wantTimeout, c.pingTimeout)
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
		pingBody              string
		bodyErr               bool
		wantErr               bool
	}{
		{
			name:                  "fails because ping url error",
			pingHandlerStatusCode: http.StatusOK,
			pingURL:               "%^*&-ERROR",
			pingBody:              regexPatternHealthcheck,
			wantErr:               true,
		},
		{
			name:                  "fails because bad response body",
			pingHandlerStatusCode: http.StatusNotFound,
			pingBody:              regexPatternHealthcheck,
			bodyErr:               true,
			wantErr:               true,
		},
		{
			name:                  "returns error because of timeout",
			pingHandlerDelay:      timeout + 1,
			pingHandlerStatusCode: http.StatusNotFound,
			pingBody:              regexPatternHealthcheck,
			wantErr:               true,
		},
		{
			name:                  "returns error from endpoint",
			pingHandlerStatusCode: http.StatusInternalServerError,
			pingBody:              regexPatternHealthcheck,
			wantErr:               true,
		},
		{
			name:                  "fails because bad response body",
			pingHandlerStatusCode: http.StatusNotFound,
			pingBody:              "error response",
			wantErr:               true,
		},
		{
			name:                  "returns success from endpoint",
			pingHandlerStatusCode: http.StatusNotFound,
			pingBody:              regexPatternHealthcheck,
			wantErr:               false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mux := http.NewServeMux()

			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if tt.pingHandlerDelay != 0 {
					time.Sleep(tt.pingHandlerDelay)
				}

				if tt.bodyErr {
					w.Header().Set("Content-Length", "1")
				}

				httputil.SendText(r.Context(), w, tt.pingHandlerStatusCode, tt.pingBody)
			})

			ts := httptest.NewServer(mux)
			defer ts.Close()

			c, err := New(
				ts.URL,
				"testorg",
				"0123456789abcdef",
				WithRetryAttempts(1),
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

func TestClient_newWriteHTTPRetrier(t *testing.T) {
	t.Parallel()

	c, err := New(
		"https://test.invalid",
		"testorg",
		"0123456789abcdef",
		WithRetryAttempts(1),
	)
	require.NoError(t, err)

	hr, err := c.newWriteHTTPRetrier()

	require.NoError(t, err)
	require.NotNil(t, hr)
}

func Test_httpRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		urlStr  string
		req     any
		wantErr bool
	}{
		{
			name:    "fail input validation",
			urlStr:  "https://test.invalid",
			req:     make(chan int), // this payload can't be encoded in JSON
			wantErr: true,
		},
		{
			name:    "fail invalid URL",
			urlStr:  "%^*&-ERROR",
			req:     make(chan int), // this payload can't be encoded in JSON
			wantErr: true,
		},
		{
			name:    "success",
			urlStr:  "https://test.invalid",
			req:     "test",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, err := httpRequest(testutil.Context(), tt.urlStr, "0123456789abcdef", tt.req)

			if !tt.wantErr {
				require.NoError(t, err)
				require.NotNil(t, r)
			} else {
				require.Error(t, err)
				require.Nil(t, r)
			}
		})
	}
}
