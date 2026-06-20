package httpreverseproxy

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
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
