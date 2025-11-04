package ipify

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
	require.Equal(t, want, c.timeout, "WithTimeout() = %want, want %want", c.timeout, want)
}

func TestWithURL(t *testing.T) {
	t.Parallel()

	want := "https://test.ipify.invalid"
	c := &Client{}
	WithURL(want)(c)
	require.Equal(t, want, c.apiURL, "WithURL() = %want, want %want", c.apiURL, want)
}

func TestWithErrorIP(t *testing.T) {
	t.Parallel()

	want := "0.0.0.0"
	c := &Client{}
	WithErrorIP(want)(c)
	require.Equal(t, want, c.errorIP, "WithErrorIP() = %want, want %want", c.errorIP, want)
}

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
