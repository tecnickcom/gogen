package sleuth

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWithTimeout(t *testing.T) {
	t.Parallel()

	want := 17 * time.Second
	c := &Client{}
	WithTimeout(want)(c)
	require.Equal(t, want, c.timeout, "WithTimeout() = %v, want %v", c.timeout, want)
}

func TestWithPingTimeout(t *testing.T) {
	t.Parallel()

	want := 23 * time.Second
	c := &Client{}
	WithPingTimeout(want)(c)
	require.Equal(t, want, c.pingTimeout, "WithPingTimeout() = %v, want %v", c.pingTimeout, want)
}

type testHTTPClient struct{}

func (thc *testHTTPClient) Do(*http.Request) (*http.Response, error) { return nil, nil }

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

func TestWithRetryDelay(t *testing.T) {
	t.Parallel()

	want := 13 * time.Second
	c := &Client{}
	WithRetryDelay(want)(c)
	require.Equal(t, want, c.retryDelay, "WithRetryDelay() = %v, want %v", c.retryDelay, want)
}
