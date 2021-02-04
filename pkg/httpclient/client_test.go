package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
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

	ctx := context.Background()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/error", nil)
	require.NoError(t, err, "failed creating http request: %v", err)

	resp, err := client.Do(req) // nolint:bodyclose
	require.Nil(t, resp)
	require.Error(t, err, "client.Do with invalud URL: an error was expected")

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	require.NoError(t, err, "failed creating http request: %v", err)

	resp, err = client.Do(req) // nolint:bodyclose
	require.NotNil(t, resp)

	defer func() { _ = resp.Body.Close() }()

	require.NoError(t, err, "client.Do(): unexpected error = %v", err)
	require.NotNil(t, resp, "returned response should not be nil")

	l, err := logging.NewLogger(logging.WithLevel(zapcore.DebugLevel))
	require.NoError(t, err, "failed creating logger: %v", err)

	ctx = logging.WithLogger(ctx, l)
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	require.NoError(t, err, "failed creating http request with context: %v", err)

	resp, err = client.Do(req) // nolint:bodyclose
	require.NotNil(t, resp)

	defer func() { _ = resp.Body.Close() }()

	require.NoError(t, err, "client.Do() with context unexpected error = %v", err)
}
