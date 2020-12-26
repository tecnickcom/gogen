//go:generate mockgen -package httputil -destination ../httputil/testutil_mock_test.go . TestHTTPResponseWriter
//go:generate mockgen -package jsendx -destination ../httputil/jsendx/testutil_mock_test.go . TestHTTPResponseWriter

package testutil

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nexmoinc/gosrvlib/pkg/httputil"
	"github.com/stretchr/testify/require"
)

func TestRouterWithHandler(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(Context(), http.MethodGet, "/test", nil)

	router := RouterWithHandler(http.MethodGet, "/test", func(w http.ResponseWriter, r *http.Request) {
		httputil.SendStatus(r.Context(), w, http.StatusOK)
	})
	router.ServeHTTP(rr, req)

	resp := rr.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
	require.Equal(t, "OK\n", string(body))
}
