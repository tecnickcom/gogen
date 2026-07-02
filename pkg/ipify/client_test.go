package ipify

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/gogen/pkg/httputil"
	"github.com/tecnickcom/gogen/pkg/testutil"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		opts        []Option
		wantTimeout time.Duration
		wantAPIURL  string
		wantErrorIP string
		wantErr     bool
	}{
		{
			name:        "succeeds with defaults",
			wantTimeout: defaultTimeout,
			wantAPIURL:  defaultAPIURL,
			wantErrorIP: defaultErrorIP,
			wantErr:     false,
		},
		{
			name: "succeeds with custom values",
			opts: []Option{
				WithTimeout(3 * time.Second),
				WithURL("http://test.ipify.invalid"),
				WithErrorIP("0.0.0.0"),
			},
			wantTimeout: 3 * time.Second,
			wantAPIURL:  "http://test.ipify.invalid",
			wantErrorIP: "0.0.0.0",
			wantErr:     false,
		},
		{
			name:        "clamps zero timeout to default",
			opts:        []Option{WithTimeout(0)},
			wantTimeout: defaultTimeout,
			wantAPIURL:  defaultAPIURL,
			wantErrorIP: defaultErrorIP,
			wantErr:     false,
		},
		{
			name:        "clamps negative timeout to default",
			opts:        []Option{WithTimeout(-1 * time.Second)},
			wantTimeout: defaultTimeout,
			wantAPIURL:  defaultAPIURL,
			wantErrorIP: defaultErrorIP,
			wantErr:     false,
		},
		{
			name:    "fails with invalid character in URL",
			opts:    []Option{WithURL("http://invalid-url.domain.invalid\u007F")},
			wantErr: true,
		},
		{
			name:    "fails with empty URL",
			opts:    []Option{WithURL("")},
			wantErr: true,
		},
		{
			name:    "fails with relative URL missing scheme and host",
			opts:    []Option{WithURL("/relative/path")},
			wantErr: true,
		},
		{
			name:    "fails with scheme but missing host",
			opts:    []Option{WithURL("http://")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := New(tt.opts...)
			if tt.wantErr {
				require.Nil(t, c, "New() returned client should be nil")
				require.Error(t, err, "New() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			require.NotNil(t, c, "New() returned client should not be nil")
			require.NoError(t, err, "New() unexpected error = %v", err)
			require.Equal(t, tt.wantTimeout, c.timeout, "New() unexpected timeout = %d got %d", tt.wantTimeout, c.timeout)
			require.Equal(t, tt.wantAPIURL, c.apiURL, "New() unexpected apiURL = %d got %d", tt.wantAPIURL, c.apiURL)
			require.Equal(t, tt.wantErrorIP, c.errorIP, "New() unexpected errorIP = %d got %d", tt.wantErrorIP, c.errorIP)
		})
	}
}

//nolint:contextcheck
func TestClient_GetPublicIP(t *testing.T) {
	t.Parallel()

	hres := httputil.NewHTTPResp(slog.Default())

	tests := []struct {
		name         string
		getIPHandler http.HandlerFunc
		wantIP       string
		wantErr      bool
	}{
		{
			name: "fails because status not OK",
			getIPHandler: func(w http.ResponseWriter, _ *http.Request) {
				hres.SendStatus(t.Context(), w, http.StatusInternalServerError)
			},
			wantIP:  "",
			wantErr: true,
		},
		{
			name: "fails because of timeout",
			getIPHandler: func(w http.ResponseWriter, _ *http.Request) {
				time.Sleep(5 * time.Second)
				hres.SendStatus(t.Context(), w, http.StatusOK)
			},
			wantErr: true,
		},
		{
			name: "fails because of bad content",
			getIPHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Length", "1")
			},
			wantErr: true,
		},
		{
			name: "succeed with valid response",
			getIPHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("0.0.0.0"))
				assert.NoError(t, err, "unexpected error: %v", err)
			},
			wantIP:  "0.0.0.0",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mux := testutil.RouterWithHandler(http.MethodGet, "/", tt.getIPHandler)
			ts := httptest.NewServer(mux)

			defer ts.Close()

			opts := []Option{WithURL(ts.URL)}
			c, err := New(opts...)

			require.NoError(t, err, "Client.GetPublicIP() create client unexpected error = %v", err)

			ip, err := c.GetPublicIP(t.Context())

			if tt.wantErr {
				require.Error(t, err, "Client.GetPublicIP() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				require.NoError(t, err, "Client.GetPublicIP() unexpected error = %v", err)
				require.Equal(t, "0.0.0.0", ip)
			}
		})
	}
}

// errorCloseHTTPClient is an [HTTPClient] returning a valid 200 response whose
// body reads successfully but fails on Close.
type errorCloseHTTPClient struct {
	body string
}

func (c *errorCloseHTTPClient) Do(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		Status:     http.StatusText(http.StatusOK),
		StatusCode: http.StatusOK,
		Body:       &errorCloser{Reader: strings.NewReader(c.body)},
	}, nil
}

// errorCloser is an io.ReadCloser whose Close always fails.
type errorCloser struct {
	*strings.Reader
}

func (e *errorCloser) Close() error {
	return errors.New("close error")
}

// TestClient_GetPublicIP_CloseError verifies the documented contract: when the
// response body Close fails after a successful read, GetPublicIP returns the
// configured errorIP together with the error, never the real IP.
func TestClient_GetPublicIP_CloseError(t *testing.T) {
	t.Parallel()

	const fallbackIP = "0.0.0.0"

	c, err := New(
		WithHTTPClient(&errorCloseHTTPClient{body: "192.0.2.1"}),
		WithErrorIP(fallbackIP),
	)
	require.NoError(t, err, "Client.GetPublicIP() create client unexpected error = %v", err)

	ip, err := c.GetPublicIP(t.Context())
	require.Error(t, err, "Client.GetPublicIP() expected close error")
	require.Equal(t, fallbackIP, ip, "on failure the configured errorIP must be returned instead of the real IP")
}

func TestClient_GetPublicIP_URLError(t *testing.T) {
	t.Parallel()

	c, err := New()
	require.NoError(t, err, "Client.GetPublicIP() create client unexpected error = %v", err)

	c.apiURL = "\x007"

	_, err = c.GetPublicIP(t.Context())
	require.Error(t, err, "Client.GetPublicIP() error = %v", err)
}
