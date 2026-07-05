package httpclient

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
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
	require.Equal(t, v, c.timeout)
	// The underlying http.Client.Timeout stays zero: the deadline is applied via
	// the request context instead, so only one timer is armed per request.
	require.Zero(t, c.client.Timeout)
}

func TestWithRoundTripper(t *testing.T) {
	t.Parallel()

	c := defaultClient()
	base := c.client.Transport
	v := func(next http.RoundTripper) http.RoundTripper { return next }
	WithRoundTripper(v)(c)
	// The identity wrapper returns its argument unchanged, so the transport is
	// still the client's own cloned transport (not http.DefaultTransport).
	require.Same(t, base, c.client.Transport)
	require.NotSame(t, http.DefaultTransport, c.client.Transport)
}

func TestWithRoundTripper_NilIgnored(t *testing.T) {
	t.Parallel()

	c := defaultClient()
	base := c.client.Transport
	WithRoundTripper(nil)(c)
	require.Same(t, base, c.client.Transport)
}

func TestWithRoundTripper_NilResultIgnored(t *testing.T) {
	t.Parallel()

	c := defaultClient()
	base := c.client.Transport
	WithRoundTripper(func(http.RoundTripper) http.RoundTripper { return nil })(c)
	require.Same(t, base, c.client.Transport)
}

func TestWithTraceIDHeaderName(t *testing.T) {
	t.Parallel()

	c := &Client{}
	v := "X-Test-Header"
	WithTraceIDHeaderName(v)(c)
	require.Equal(t, v, c.traceIDHeaderName)
}

func TestWithTraceIDHeaderName_EmptyIgnored(t *testing.T) {
	t.Parallel()

	c := &Client{traceIDHeaderName: "X-Keep"}
	WithTraceIDHeaderName("")(c)
	require.Equal(t, "X-Keep", c.traceIDHeaderName)
}

func TestWithComponent(t *testing.T) {
	t.Parallel()

	c := &Client{}
	v := "test_123"
	WithComponent(v)(c)
	require.Equal(t, v, c.component)
}

func TestWithComponent_EmptyIgnored(t *testing.T) {
	t.Parallel()

	c := &Client{component: "-"}
	WithComponent("")(c)
	require.Equal(t, "-", c.component)
}

func TestWithRedactFn(t *testing.T) {
	t.Parallel()

	c := &Client{}
	v := func(b []byte) string { return string(b) + "test" }
	WithRedactFn(v)(c)
	require.Equal(t, "alphatest", c.redactFn([]byte("alpha")))
}

func TestWithRedactFn_NilIgnored(t *testing.T) {
	t.Parallel()

	keep := func(b []byte) string { return string(b) + "keep" }
	c := &Client{redactFn: keep}
	WithRedactFn(nil)(c)
	require.Equal(t, "xkeep", c.redactFn([]byte("x")))
}

func TestWithLogPrefix(t *testing.T) {
	t.Parallel()

	c := &Client{}
	v := "prefixtest_"
	WithLogPrefix(v)(c)
	require.Equal(t, v, c.logPrefix)
}

func TestWithMaxDumpSize(t *testing.T) {
	t.Parallel()

	c := defaultClient()
	WithMaxDumpSize(42)(c)
	require.Equal(t, int64(42), c.maxDumpSize)
}

func TestWithDialContext(t *testing.T) {
	t.Parallel()

	c := defaultClient()
	v := func(_ context.Context, _, _ string) (net.Conn, error) { return nil, errors.New("TEST") }
	WithDialContext(v)(c)

	tr, ok := c.client.Transport.(*http.Transport)
	require.True(t, ok)

	out, err := tr.DialContext(t.Context(), "", "")

	require.Error(t, err)
	require.Nil(t, out)
}

func TestWithDialContext_NilIgnored(t *testing.T) {
	t.Parallel()

	c := defaultClient()
	tr := c.client.Transport.(*http.Transport) //nolint:forcetypeassert
	base := tr.DialContext

	WithDialContext(nil)(c)
	// The dialer is unchanged (still the clone's default), so no panic and no
	// accidental reset.
	require.Equal(t, base == nil, tr.DialContext == nil)
}

func TestWithLogger(t *testing.T) {
	t.Parallel()

	c := defaultClient()
	v := slog.Default()
	WithLogger(v)(c)
	require.Equal(t, v, c.logger)
}

func TestWithLogger_NilIgnored(t *testing.T) {
	t.Parallel()

	keep := slog.Default()
	c := &Client{logger: keep}
	WithLogger(nil)(c)
	require.Same(t, keep, c.logger)
}

func TestWithTransport(t *testing.T) {
	t.Parallel()

	c := defaultClient()
	custom := &http.Transport{MaxIdleConnsPerHost: 7}
	WithTransport(custom)(c)

	tr, ok := c.client.Transport.(*http.Transport)
	require.True(t, ok)
	require.Equal(t, 7, tr.MaxIdleConnsPerHost)

	// The transport is cloned: it is a distinct object and later mutations of the
	// caller's transport do not affect the client.
	require.NotSame(t, custom, tr)
	custom.MaxIdleConnsPerHost = 99
	require.Equal(t, 7, tr.MaxIdleConnsPerHost)
}

func TestWithTransport_NilIgnored(t *testing.T) {
	t.Parallel()

	c := defaultClient()
	base := c.client.Transport
	WithTransport(nil)(c)
	require.Same(t, base, c.client.Transport)
}

func TestWithTLSClientConfig(t *testing.T) {
	t.Parallel()

	cfg := &tls.Config{MinVersion: tls.VersionTLS13}
	c := defaultClient()
	WithTLSClientConfig(cfg)(c)

	tr, ok := c.client.Transport.(*http.Transport)
	require.True(t, ok)
	require.Same(t, cfg, tr.TLSClientConfig)
}

func TestWithTLSClientConfig_NilIgnored(t *testing.T) {
	t.Parallel()

	keep := &tls.Config{MinVersion: tls.VersionTLS12}
	c := defaultClient()
	WithTLSClientConfig(keep)(c)
	WithTLSClientConfig(nil)(c) // nil must not clear the previously set config

	tr, ok := c.client.Transport.(*http.Transport)
	require.True(t, ok)
	require.Same(t, keep, tr.TLSClientConfig)
}

func TestWithTLSClientConfig_AfterRoundTripperNoOp(t *testing.T) {
	t.Parallel()

	wrap := func(next http.RoundTripper) http.RoundTripper {
		return &passthroughRoundTripper{next: next}
	}
	cfg := &tls.Config{MinVersion: tls.VersionTLS13}

	// WithRoundTripper wraps first, so the transport is no longer *http.Transport
	// and the later WithTLSClientConfig is a silent no-op.
	c := New(WithRoundTripper(wrap), WithTLSClientConfig(cfg))

	rt, ok := c.client.Transport.(*passthroughRoundTripper)
	require.True(t, ok)

	inner, ok := rt.next.(*http.Transport)
	require.True(t, ok)
	require.NotSame(t, cfg, inner.TLSClientConfig,
		"WithTLSClientConfig after WithRoundTripper must not take effect")
}
