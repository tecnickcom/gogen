package ipify

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nexmoinc/gosrvlib/pkg/httputil"
	"github.com/nexmoinc/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		opts        []ClientOption
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
			opts: []ClientOption{
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
			opts:    []ClientOption{WithURL("http://invalid-url.domain.invalid\u007F")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := NewClient(tt.opts...)
			if tt.wantErr {
				require.Nil(t, c, "NewClient() returned client should be nil")
				require.Error(t, err, "NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.NotNil(t, c, "NewClient() returned client should not be nil")
			require.NoError(t, err, "NewClient() unexpected error = %v", err)
			require.Equal(t, tt.wantTimeout, c.timeout, "NewClient() unexpected timeout = %d got %d", tt.wantTimeout, c.timeout)
			require.Equal(t, tt.wantAPIURL, c.apiURL, "NewClient() unexpected apiURL = %d got %d", tt.wantAPIURL, c.apiURL)
			require.Equal(t, tt.wantErrorIP, c.errorIP, "NewClient() unexpected errorIP = %d got %d", tt.wantErrorIP, c.errorIP)
		})
	}
}

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

			opts := []ClientOption{WithURL(ts.URL)}
			c, err := NewClient(opts...)
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

	c, err := NewClient()
	require.NoError(t, err, "Client.GetPublicIP() create client unexpected error = %v", err)

	c.apiURL = "\x007"

	_, err = c.GetPublicIP(testutil.Context())
	require.Error(t, err, "Client.GetPublicIP() error = %v", err)
}
