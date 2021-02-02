package httpclient

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()
	timeout := 17 * time.Second
	traceid := "test-header-123"
	component := "test-component"
	fn := func(next http.RoundTripper) http.RoundTripper { return next }
	opts := []Option{
		WithTimeout(timeout),
		WithRoundTripper(fn),
		WithTraceIDHeaderName(traceid),
		WithComponent(component),
	}
	got := New(opts...)
	require.NotNil(t, got, "New() returned client should not be nil")
	require.Equal(t, traceid, got.traceIDHeaderName)
	require.Equal(t, component, got.component)
	require.Equal(t, timeout, got.client.Timeout)
	require.Equal(t, fn(http.DefaultTransport), got.client.Transport)
}

func TestDo(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`OK`))
	}))
	defer server.Close()

	client := New()

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	require.NoError(t, err, "failed creating http request: %s", err)
	resp, err := client.Do(req)
	require.NoError(t, err, "client.Do(): unexpected error = %v", err)
	require.NotNil(t, resp, "returned response should not be nil")

	req, err = http.NewRequest(http.MethodGet, "/error", nil)
	require.NoError(t, err, "failed creating http request: %s", err)
	_, err = client.Do(req)
	require.Error(t, err, "client.Do with invalud URL: an error was expected")
}
