package ipify

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
			name:    "fails with invalid character in URL",
			opts:    []Option{WithURL("http://invalid-url.domain.invalid\u007F")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
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

	tests := []struct {
		name         string
		getIPHandler http.HandlerFunc
		wantIP       string
		wantErr      bool
	}{
		{
			name: "fails because status not OK",
			getIPHandler: func(w http.ResponseWriter, r *http.Request) {
				httputil.SendStatus(testutil.Context(), w, http.StatusInternalServerError)
			},
			wantIP:  "",
			wantErr: true,
		},
		{
			name: "fails because of timeout",
			getIPHandler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(5 * time.Second)
				httputil.SendStatus(testutil.Context(), w, http.StatusOK)
			},
			wantErr: true,
		},
		{
			name: "fails because of bad content",
			getIPHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", "1")
			},
			wantErr: true,
		},
		{
			name: "succeed with valid response",
			getIPHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("0.0.0.0"))
				require.NoError(t, err, "unexpected error: %v", err)
			},
			wantIP:  "0.0.0.0",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mux := testutil.RouterWithHandler(http.MethodGet, "/", tt.getIPHandler)
			ts := httptest.NewServer(mux)
			defer ts.Close()

			opts := []Option{WithURL(ts.URL)}
			c, err := New(opts...)
			require.NoError(t, err, "Client.GetPublicIP() create client unexpected error = %v", err)

			ip, err := c.GetPublicIP(testutil.Context())

			if tt.wantErr {
				require.Error(t, err, "Client.GetPublicIP() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				require.Nil(t, err, "Client.GetPublicIP() unexpected error = %v", err)
				require.Equal(t, "0.0.0.0", ip)
			}
		})
	}
}

func TestClient_GetPublicIP_URLError(t *testing.T) {
	t.Parallel()

	c, err := New()
	require.NoError(t, err, "Client.GetPublicIP() create client unexpected error = %v", err)

	c.apiURL = "\x007"

	_, err = c.GetPublicIP(testutil.Context())
	require.Error(t, err, "Client.GetPublicIP() error = %v", err)
}
