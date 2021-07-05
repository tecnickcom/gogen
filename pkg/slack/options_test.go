package slack

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWithRequestTimeout(t *testing.T) {
	t.Parallel()

	want := 17 * time.Second
	c := &Client{}
	WithRequestTimeout(want)(c)
	require.Equal(t, want, c.requestTimeout, "WithTimeout() = %v, want %v", c.requestTimeout, want)
}

func TestWithPingTimeout(t *testing.T) {
	t.Parallel()

	want := 23 * time.Second
	c := &Client{}
	WithPingTimeout(want)(c)
	require.Equal(t, want, c.pingTimeout, "WithPingTimeout() = %v, want %v", c.pingTimeout, want)
}

func TestWithPingURL(t *testing.T) {
	t.Parallel()

	want := "https://test.ping.url.invalid"
	c := &Client{}
	WithPingURL(want)(c)
	require.Equal(t, want, c.pingURL, "WithPingURL() = %v, want %v", c.pingURL, want)
}

type testHTTPClient struct{}

func (thc *testHTTPClient) Do(r *http.Request) (*http.Response, error) { return nil, nil }

func TestWithHTTPClient(t *testing.T) {
	t.Parallel()

	v := &testHTTPClient{}
	c := &Client{}
	WithHTTPClient(v)(c)
	require.Equal(t, reflect.ValueOf(v).Pointer(), reflect.ValueOf(c.httpClient).Pointer())
}

func TestWithRetryAttempts(t *testing.T) {
	t.Parallel()

	v := uint(3)
	c := &Client{}
	WithRetryAttempts(v)(c)
	require.Equal(t, v, c.retryAttempts, "WithRetryAttempts() = %v, want %v", c.retryAttempts, v)
}
