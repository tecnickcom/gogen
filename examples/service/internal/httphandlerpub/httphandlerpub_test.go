package httphandlerpub

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	hh := New(nil, nil)
	require.NotNil(t, hh)
}

func TestHTTPHandlerPublic_BindHTTP(t *testing.T) {
	t.Parallel()

	h := &HTTPHandlerPublic{}
	got := h.BindHTTP(t.Context())
	require.Len(t, got, 1)
}

func TestHTTPHandlerPublic_handleGenUID(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

	hh := New(nil, nil)
	require.NotNil(t, hh)

	hh.handleGenUID(rr, req)

	resp := rr.Result()
	require.NotNil(t, resp)

	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err, "error closing resp.Body")
	}()

	body, _ := io.ReadAll(resp.Body)

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	require.NotEmpty(t, string(body))
}
