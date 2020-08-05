// +build unit

package httputil_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/nexmoinc/gosrvlib/pkg/httputil"
	"github.com/stretchr/testify/require"
)

func TestHeaderOrDefault(t *testing.T) {
	t.Parallel()

	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	r.Header.Add("set-header", "test")

	v1 := httputil.HeaderOrDefault(r, "unset-header", "default")
	require.Equal(t, "default", v1)

	v2 := httputil.HeaderOrDefault(r, "set-header", "default")
	require.Equal(t, "test", v2)
}

func TestPathParam(t *testing.T) {
	r := httprouter.New()

	r.HandlerFunc(http.MethodGet, "/resource/*id", func(w http.ResponseWriter, r *http.Request) {
		id := httputil.PathParam(r, "id")
		httputil.SendText(r.Context(), w, http.StatusOK, id)
	})

	pathID := "id-12345"

	rr := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/resource/"+pathID, nil)
	require.NoError(t, err)

	r.ServeHTTP(rr, req)

	body, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	require.Equal(t, pathID, string(body))
}
