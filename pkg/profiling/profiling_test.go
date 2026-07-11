package profiling

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/nurago/pkg/testutil"
)

func TestPProfHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		path         string
		wantStatus   int
		wantCT       string   // optional Content-Type substring; "" skips the check
		wantBody     []string // optional substrings that must appear in the body
		wantNonEmpty bool     // require a non-empty body (for binary profiles)
	}{
		{
			name:       "index lists profiles with relative links",
			path:       "/pprof/",
			wantStatus: http.StatusOK,
			wantCT:     "text/html",
			// The relative href (no leading slash) is what makes the index work
			// under any mount prefix; assert it explicitly.
			wantBody: []string{"Types of profiles available", "href='heap?debug=1'"},
		},
		{
			name:         "cmdline",
			path:         "/pprof/cmdline",
			wantStatus:   http.StatusOK,
			wantCT:       "text/plain",
			wantNonEmpty: true,
		},
		{
			name:         "cpu profile",
			path:         "/pprof/profile?seconds=1",
			wantStatus:   http.StatusOK,
			wantCT:       "application/octet-stream",
			wantNonEmpty: true,
		},
		{
			name:       "symbol",
			path:       "/pprof/symbol",
			wantStatus: http.StatusOK,
			wantCT:     "text/plain",
			wantBody:   []string{"num_symbols:"},
		},
		{
			name:         "trace",
			path:         "/pprof/trace",
			wantStatus:   http.StatusOK,
			wantCT:       "application/octet-stream",
			wantNonEmpty: true,
		},
		{
			name:         "heap profile via named handler",
			path:         "/pprof/heap",
			wantStatus:   http.StatusOK,
			wantNonEmpty: true,
		},
		{
			name:         "trailing slash on a named profile still resolves",
			path:         "/pprof/heap/",
			wantStatus:   http.StatusOK,
			wantNonEmpty: true,
		},
		{
			name:       "unknown profile returns 404",
			path:       "/pprof/does-not-exist",
			wantStatus: http.StatusNotFound,
			wantBody:   []string{"Unknown profile"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := testutil.RouterWithHandler(http.MethodGet, "/pprof/*"+WildcardParamName, PProfHandler)

			rr := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tt.path, nil)

			// ServeHTTP populates the httprouter params in the request context,
			// exercising the same path extraction as a live request.
			r.ServeHTTP(rr, req)

			require.Equal(t, tt.wantStatus, rr.Code, "unexpected status for path %q", tt.path)

			if tt.wantCT != "" {
				require.Contains(t, rr.Header().Get("Content-Type"), tt.wantCT,
					"unexpected Content-Type for path %q", tt.path)
			}

			body := rr.Body.String()
			for _, want := range tt.wantBody {
				require.Contains(t, body, want, "missing %q in body for path %q", want, tt.path)
			}

			if tt.wantNonEmpty {
				require.NotEmpty(t, body, "expected a non-empty body for path %q", tt.path)
			}
		})
	}
}

// TestPProfHandlerWithoutRouterParams verifies the handler is defensive: invoked
// outside httprouter (no params in the request context), it must not panic and
// should fall back to serving the index page.
func TestPProfHandlerWithoutRouterParams(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/pprof", nil)

	PProfHandler(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Header().Get("Content-Type"), "text/html")
	require.Contains(t, rr.Body.String(), "Types of profiles available")
}

// TestPProfHandlerEndToEnd exercises the full network round-trip through a real
// server. A request without a trailing slash must be redirected to /pprof/
// (httprouter's default RedirectTrailingSlash) so the index page's relative
// links resolve correctly.
func TestPProfHandlerEndToEnd(t *testing.T) {
	t.Parallel()

	r := testutil.RouterWithHandler(http.MethodGet, "/pprof/*"+WildcardParamName, PProfHandler)

	ts := httptest.NewServer(r)
	defer ts.Close()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+"/pprof", nil)
	require.NoError(t, err, "unexpected error while creating request")

	client := &http.Client{Timeout: 2 * time.Second}

	resp, err := client.Do(req)
	require.NoError(t, err, "unexpected error while performing request %q", req.URL.String())
	require.NotNil(t, resp)

	defer func() {
		require.NoError(t, resp.Body.Close(), "error closing resp.Body")
	}()

	require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d", resp.StatusCode)
	require.Contains(t, resp.Header.Get("Content-Type"), "text/html")
}
