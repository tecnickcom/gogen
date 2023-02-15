//go:generate mockgen -package httputil -destination ../httputil/testutil_mock_test.go . TestHTTPResponseWriter
//go:generate mockgen -package jsendx -destination ../httputil/jsendx/testutil_mock_test.go . TestHTTPResponseWriter

package testutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Vonage/gosrvlib/pkg/httputil"
	"github.com/stretchr/testify/require"
)

func TestRouterWithHandler(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(Context(), http.MethodGet, "/test", nil)

	router := RouterWithHandler(http.MethodGet, "/test", func(w http.ResponseWriter, r *http.Request) {
		httputil.SendStatus(r.Context(), w, http.StatusOK)
	})
	router.ServeHTTP(rr, req)

	resp := rr.Result() //nolint:bodyclose
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()

	body, _ := io.ReadAll(resp.Body)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Equal(t, "OK\n", string(body))
}
