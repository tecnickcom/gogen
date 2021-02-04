package httpclient

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWithTimeout(t *testing.T) {
	t.Parallel()

	c := defaultClient()
	v := 13 * time.Second
	WithTimeout(v)(c)
	require.Equal(t, v, c.client.Timeout)
}

func TestWithRoundTripper(t *testing.T) {
	t.Parallel()

	c := defaultClient()
	v := func(next http.RoundTripper) http.RoundTripper { return next }
	WithRoundTripper(v)(c)
	require.Equal(t, v(http.DefaultTransport), c.client.Transport)
}

func TestWithTraceIDHeaderName(t *testing.T) {
	t.Parallel()

	c := &Client{}
	v := "X-Test-Header"
	WithTraceIDHeaderName(v)(c)
	require.Equal(t, v, c.traceIDHeaderName)
}

func TestWithComponent(t *testing.T) {
	t.Parallel()

	c := &Client{}
	v := "test_123"
	WithComponent(v)(c)
	require.Equal(t, v, c.component)
}
