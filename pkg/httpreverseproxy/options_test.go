package httpreverseproxy

import (
	"net/http"
	"net/http/httputil"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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

	l := zap.NewNop()
	c := &Client{}
	WithLogger(l)(c)
	require.Equal(t, reflect.ValueOf(l).Pointer(), reflect.ValueOf(c.logger).Pointer())
}
