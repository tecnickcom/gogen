package httpreverseproxy

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/nurago/pkg/redact"
)

type testHTTPClient struct{}

func (thc *testHTTPClient) Do(_ *http.Request) (*http.Response, error) {
	return nil, nil //nolint:nilnil
}

func TestWithHTTPClient(t *testing.T) {
	t.Parallel()

	v := &testHTTPClient{}
	c := &Client{}
	WithHTTPClient(v)(c)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(c.httpClient).Pointer())
}

func TestWithReverseProxy(t *testing.T) {
	t.Parallel()

	v := &httputil.ReverseProxy{}
	c := &Client{}
	WithReverseProxy(v)(c)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(c.proxy).Pointer())
}

func TestWithLogger(t *testing.T) {
	t.Parallel()

	l := slog.Default()
	c := &Client{}
	WithLogger(l)(c)
	require.Equal(t, l, c.logger)
}

func TestWithPathParam(t *testing.T) {
	t.Parallel()

	c := &Client{}
	WithPathParam("upstream")(c)
	require.Equal(t, "upstream", c.pathParam)

	// An empty name falls back to the default.
	WithPathParam("")(c)
	require.Equal(t, defaultPathParam, c.pathParam)
}

func TestWithRedactFn(t *testing.T) {
	t.Parallel()

	sentinel := func(_ []byte) string { return "REDACTED" }
	c := &Client{redactFn: redact.HTTPDataString}

	WithRedactFn(sentinel)(c)
	require.Equal(t, "REDACTED", c.redactFn(nil))

	// A nil fn is ignored and keeps the previously set function.
	WithRedactFn(nil)(c)
	require.Equal(t, "REDACTED", c.redactFn(nil))
}

func TestWithLaxBasePath(t *testing.T) {
	t.Parallel()

	c := &Client{}
	require.False(t, c.laxBasePath, "base-path check is strict by default")

	WithLaxBasePath()(c)
	require.True(t, c.laxBasePath)
}
