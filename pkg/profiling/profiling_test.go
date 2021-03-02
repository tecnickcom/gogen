package profiling

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nexmoinc/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func TestPProfHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
	}{
		{
			name: "success with /pprof index",
			path: "/pprof",
		},
		{
			name: "success with /pprof/cmdline index",
			path: "/pprof/cmdline",
		},
		{
			name: "success with /pprof/profile index",
			path: "/pprof/profile?seconds=1",
		},
		{
			name: "success with /pprof/symbol index",
			path: "/pprof/symbol",
		},
		{
			name: "success with /pprof/trace index",
			path: "/pprof/trace",
		},
		{
			name: "success with /pprof/heap index",
			path: "/pprof/heap",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := testutil.RouterWithHandler(http.MethodGet, "/pprof/*option", PProfHandler)

			ts := httptest.NewServer(r)
			defer ts.Close()

			ctx := context.Background()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s%s", ts.URL, tt.path), nil)
			require.NoError(t, err, "unexpected error while creating request for path %q", tt.path)

			var testHTTPClient = &http.Client{Timeout: 2 * time.Second}

			resp, err := testHTTPClient.Do(req)
			require.NoError(t, err, "unexpected error while performing request %q", req.URL.String())
			require.NotNil(t, resp)

			defer func() {
				err := resp.Body.Close()
				require.NoError(t, err, "error closing resp.Body")
			}()

			require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected status code %d", resp.StatusCode)
		})
	}
}
