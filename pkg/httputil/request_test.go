// +build unit

package httputil

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nexmoinc/gosrvlib/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func TestHeaderOrDefault(t *testing.T) {
	t.Parallel()

	r, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	r.Header.Add("set-header", "test")

	v1 := HeaderOrDefault(r, "unset-header", "default")
	require.Equal(t, "default", v1)

	v2 := HeaderOrDefault(r, "set-header", "default")
	require.Equal(t, "test", v2)
}

func TestPathParam(t *testing.T) {
	tests := []struct {
		name        string
		mappedPath  string
		paramName   string
		requestPath string
		wantBody    string
	}{
		{
			name:        "returns empty value with invalid param name",
			mappedPath:  "/resource/*id",
			paramName:   "invalid",
			requestPath: "/resource/test-12345",
			wantBody:    "",
		},
		{
			name:        "succeed",
			mappedPath:  "/resource/*id",
			paramName:   "id",
			requestPath: "/resource/test-12345",
			wantBody:    "test-12345",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := testutil.RouterWithHandler(http.MethodGet, tt.mappedPath, func(w http.ResponseWriter, r *http.Request) {
				val := PathParam(r, tt.paramName)
				SendText(r.Context(), w, http.StatusOK, val)
			})

			rr := httptest.NewRecorder()
			req, err := http.NewRequest("GET", tt.requestPath, nil)
			require.NoError(t, err)

			r.ServeHTTP(rr, req)

			body, err := ioutil.ReadAll(rr.Body)
			require.NoError(t, err)

			require.Equal(t, http.StatusOK, rr.Code)
			require.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			require.Equal(t, tt.wantBody, string(body))
		})
	}
}

func TestAddBasicAuth(t *testing.T) {
	t.Parallel()

	r, _ := http.NewRequest(http.MethodGet, "", nil)
	AddBasicAuth("key", "secret", r)

	wanted, _ := http.NewRequest(http.MethodGet, "", nil)
	wanted.Header.Set("Authorization", "Basic a2V5OnNlY3JldA==")
	require.Equal(t, r, wanted)
}
