package httpreverseproxy

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	libhttputil "github.com/tecnickcom/gogen/pkg/httputil"
	"github.com/tecnickcom/gogen/pkg/testutil"
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
			opts:        []Option{WithLogger(slog.Default())},
			wantErr:     false,
		},
		{
			name:        "succeeds with custom path param",
			serviceAddr: "http://service.domain.invalid:1238/",
			opts:        []Option{WithPathParam("upstream")},
			wantErr:     false,
		},
		{
			name:        "succeeds with empty path param falling back to default",
			serviceAddr: "http://service.domain.invalid:1239/",
			opts:        []Option{WithPathParam("")},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
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

	hres := libhttputil.NewHTTPResp(slog.Default())

	targetMux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			hres.SendStatus(r.Context(), w, http.StatusOK)
		}()

		rd, err := httputil.DumpRequest(r, false)
		assert.NoError(t, err)
		t.Logf("%s", string(rd))

		proxyTestURL, err := url.Parse(targetServer.URL)
		assert.NoError(t, err)

		assert.Equal(t, r.Host, proxyTestURL.Host)
		assert.Equal(t, "127.0.0.1", r.Header.Get("X-Forwarded-For"))
	})

	targetMux.HandleFunc("/badrequest", func(w http.ResponseWriter, r *http.Request) {
		hres.SendStatus(r.Context(), w, http.StatusBadRequest)
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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := []Option{}
			if tt.withLogger {
				opts = append(opts, WithLogger(slog.Default()))
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

			ctx := t.Context()

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

// TestClient_ForwardRequest_PreservesBasePath verifies that the base path of
// the configured upstream address (e.g. "/v2") is preserved by the default
// rewrite instead of being clobbered by the catch-all path parameter.
func TestClient_ForwardRequest_PreservesBasePath(t *testing.T) {
	t.Parallel()

	const timeout = 1 * time.Second

	targetMux := http.NewServeMux()
	targetServer := httptest.NewServer(targetMux)

	t.Cleanup(func() { targetServer.Close() })

	hres := libhttputil.NewHTTPResp(slog.Default())

	targetMux.HandleFunc("/v2/users", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/users", r.URL.Path)
		hres.SendStatus(r.Context(), w, http.StatusOK)
	})

	// The upstream address carries a base path that must survive the rewrite.
	c, err := New(targetServer.URL + "/v2")
	require.NoError(t, err)

	proxyMux := testutil.RouterWithHandler(http.MethodGet, "/proxy/*path", c.ForwardRequest)
	proxyServer := httptest.NewServer(proxyMux)

	t.Cleanup(func() { proxyServer.Close() })

	ctx := t.Context()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, proxyServer.URL+"/proxy/users", nil)

	hc := &http.Client{Timeout: timeout}
	resp, err := hc.Do(req)
	require.NoError(t, err)

	t.Cleanup(func() {
		if resp != nil {
			assert.NoError(t, resp.Body.Close())
		}
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestClient_ForwardRequest_ForwardsRedirect verifies that the default
// upstream client does not follow 3xx responses: the redirect must be
// forwarded verbatim to the caller instead of being fetched by the proxy.
func TestClient_ForwardRequest_ForwardsRedirect(t *testing.T) {
	t.Parallel()

	const timeout = 1 * time.Second

	targetMux := http.NewServeMux()
	targetServer := httptest.NewServer(targetMux)

	t.Cleanup(func() { targetServer.Close() })

	targetMux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/redirected", http.StatusFound)
	})

	targetMux.HandleFunc("/redirected", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("FINAL"))
	})

	c, err := New(targetServer.URL)
	require.NoError(t, err)

	proxyMux := testutil.RouterWithHandler(http.MethodGet, "/proxy/*path", c.ForwardRequest)
	proxyServer := httptest.NewServer(proxyMux)

	t.Cleanup(func() { proxyServer.Close() })

	ctx := t.Context()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, proxyServer.URL+"/proxy/redirect", nil)

	// The test client must not follow redirects either, so it observes the
	// exact status the proxy sent back.
	hc := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := hc.Do(req)
	require.NoError(t, err)

	t.Cleanup(func() {
		if resp != nil {
			assert.NoError(t, resp.Body.Close())
		}
	})

	require.Equal(t, http.StatusFound, resp.StatusCode, "the proxy must forward the 3xx instead of following it")
	require.Equal(t, "/redirected", resp.Header.Get("Location"))
}

func TestClient_ForwardRequest_CustomPathParam(t *testing.T) {
	t.Parallel()

	const timeout = 1 * time.Second

	targetMux := http.NewServeMux()
	targetServer := httptest.NewServer(targetMux)

	t.Cleanup(func() { targetServer.Close() })

	hres := libhttputil.NewHTTPResp(slog.Default())

	targetMux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		// The upstream path must be preserved through the custom wildcard param.
		assert.Equal(t, "/test", r.URL.Path)
		hres.SendStatus(r.Context(), w, http.StatusOK)
	})

	// Register the catch-all under a non-default param name and configure it.
	c, err := New(targetServer.URL, WithPathParam("upstream"))
	require.NoError(t, err)

	proxyMux := testutil.RouterWithHandler(http.MethodGet, "/proxy/*upstream", c.ForwardRequest)
	proxyServer := httptest.NewServer(proxyMux)

	t.Cleanup(func() { proxyServer.Close() })

	ctx := t.Context()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, proxyServer.URL+"/proxy/test", nil)

	hc := &http.Client{Timeout: timeout}
	resp, err := hc.Do(req)
	require.NoError(t, err)

	t.Cleanup(func() {
		if resp != nil {
			assert.NoError(t, resp.Body.Close())
		}
	})

	require.Equal(t, http.StatusOK, resp.StatusCode)
}
