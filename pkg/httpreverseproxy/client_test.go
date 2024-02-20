package httpreverseproxy

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"
	"time"

	libhttputil "github.com/Vonage/gosrvlib/pkg/httputil"
	"github.com/Vonage/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		serviceAddr string
		opts        []Option
		wantErr     bool
	}{
		{
			name:        "fails with invalid character in URL",
			serviceAddr: "http://invalid-url.domain.invalid\u007F",
			wantErr:     true,
		},
		{
			name:        "succeeds with defaults",
			serviceAddr: "http://service.domain.invalid:1234/",
			wantErr:     false,
		},
		{
			name:        "succeeds with custom http client",
			serviceAddr: "http://service.domain.invalid:1235/",
			opts:        []Option{WithHTTPClient(&testHTTPClient{})},
			wantErr:     false,
		},
		{
			name:        "succeeds with custom reverse proxy",
			serviceAddr: "http://service.domain.invalid:1236/",
			opts:        []Option{WithReverseProxy(&httputil.ReverseProxy{})},
			wantErr:     false,
		},
		{
			name:        "succeeds with custom logger",
			serviceAddr: "http://service.domain.invalid:1237/",
			opts:        []Option{WithLogger(zap.NewNop())},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := New(tt.serviceAddr, tt.opts...)
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
func TestClient_ForwardRequest(t *testing.T) {
	t.Parallel()

	const timeout = 1 * time.Second

	// setup target test server
	targetMux := http.NewServeMux()

	targetServer := httptest.NewServer(targetMux)

	t.Cleanup(
		func() {
			targetServer.Close()
		},
	)

	targetMux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			libhttputil.SendStatus(r.Context(), w, http.StatusOK)
		}()

		rd, err := httputil.DumpRequest(r, false)
		require.NoError(t, err)
		t.Logf("%s", string(rd))

		proxyTestURL, err := url.Parse(targetServer.URL)
		require.NoError(t, err)

		require.Equal(t, r.Host, proxyTestURL.Host)
		require.Equal(t, "127.0.0.1", r.Header.Get("X-Forwarded-For"))
	})

	targetMux.HandleFunc("/badrequest", func(w http.ResponseWriter, r *http.Request) {
		libhttputil.SendStatus(r.Context(), w, http.StatusBadRequest)
	})

	targetMux.HandleFunc("/error", func(_ http.ResponseWriter, _ *http.Request) {
		time.Sleep(1 + timeout)
	})

	tests := []struct {
		name       string
		path       string
		status     int
		withLogger bool
		wantErr    bool
	}{
		{
			name:   "success OK",
			path:   "/proxy/test",
			status: http.StatusOK,
		},
		{
			name:   "Not Found",
			path:   "/proxy/notfound",
			status: http.StatusNotFound,
		},
		{
			name:   "Bad Request",
			path:   "/proxy/badrequest",
			status: http.StatusBadRequest,
		},
		{
			name:    "Backend Error",
			path:    "/proxy/error",
			wantErr: true,
		},
		{
			name:       "Backend Error with logger",
			path:       "/proxy/error",
			withLogger: true,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := []Option{}
			if tt.withLogger {
				opts = append(opts, WithLogger(zap.NewNop()))
			}

			// setup proxy test server
			c, err := New(targetServer.URL, opts...)
			require.NoError(t, err)

			proxyMux := testutil.RouterWithHandler(http.MethodGet, "/proxy/*path", c.ForwardRequest)

			proxyServer := httptest.NewServer(proxyMux)

			t.Cleanup(
				func() {
					proxyServer.Close()
				},
			)

			ctx := testutil.Context()

			// perform test
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, proxyServer.URL+tt.path, nil)

			hc := &http.Client{Timeout: timeout}
			resp, err := hc.Do(req)

			t.Cleanup(
				func() {
					if resp != nil {
						err := resp.Body.Close()
						require.NoError(t, err)
					}
				},
			)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.status, resp.StatusCode)
			}
		})
	}
}
