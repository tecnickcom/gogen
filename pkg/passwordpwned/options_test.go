package passwordpwned

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWithURL(t *testing.T) {
	t.Parallel()

	want := "https://test.haveibeenpwned.invalid"
	c := &Client{}
	WithURL(want)(c)
	require.Equal(t, want, c.apiURL, "WithURL() = %want, want %want", c.apiURL, want)
}

func TestWithUserAgent(t *testing.T) {
	t.Parallel()

	want := "test.user.agent/3"
	c := &Client{}
	WithUserAgent(want)(c)
	require.Equal(t, want, c.userAgent, "WithUserAgent() = %want, want %want", c.userAgent, want)
}

func TestWithTimeout(t *testing.T) {
	t.Parallel()

	want := 17 * time.Second
	c := &Client{}
	WithTimeout(want)(c)
	require.Equal(t, want, c.timeout, "WithTimeout() = %v, want %v", c.timeout, want)
}

type testHTTPClient struct{}

func (thc *testHTTPClient) Do(*http.Request) (*http.Response, error) {
	return nil, nil //nolint:nilnil
}

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
