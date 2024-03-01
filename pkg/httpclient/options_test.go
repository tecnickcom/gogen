package httpclient

import (
	"context"
	"errors"
	"net"
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

func TestWithRedactFn(t *testing.T) {
	t.Parallel()

	c := &Client{}
	v := func(s string) string { return s + "test" }
	WithRedactFn(v)(c)
	require.Equal(t, "alphatest", c.redactFn("alpha"))
}

func TestWithLogPrefix(t *testing.T) {
	t.Parallel()

	c := &Client{}
	v := "prefixtest_"
	WithLogPrefix(v)(c)
	require.Equal(t, v, c.logPrefix)
}

func TestWithDialContext(t *testing.T) {
	t.Parallel()

	c := defaultClient()
	v := func(_ context.Context, _, _ string) (net.Conn, error) { return nil, errors.New("TEST") }
	WithDialContext(v)(c)

	out, err := c.client.Transport.(*http.Transport).DialContext(context.TODO(), "", "")
	require.Error(t, err)
	require.Nil(t, out)
}
