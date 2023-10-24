package httpretrier

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWithRetryIfFn(t *testing.T) {
	t.Parallel()

	c := &HTTPRetrier{}

	v := func(r *http.Response, err error) bool { return true }
	err := WithRetryIfFn(v)(c)
	require.NoError(t, err)

	resp := &http.Response{
		Status:     http.StatusText(http.StatusOK),
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader([]byte{})),
	}
	require.True(t, c.retryIfFn(resp, nil))

	v = nil
	err = WithRetryIfFn(v)(c)
	require.Error(t, err)
}

func TestWithAttempts(t *testing.T) {
	t.Parallel()

	var v uint

	c := defaultHTTPRetrier()

	v = 5
	err := WithAttempts(v)(c)
	require.NoError(t, err)
	require.Equal(t, v, c.attempts)

	v = 0
	err = WithAttempts(v)(c)
	require.Error(t, err)
}

func TestWithDelay(t *testing.T) {
	t.Parallel()

	var v time.Duration

	c := defaultHTTPRetrier()

	v = 503 * time.Millisecond
	err := WithDelay(v)(c)
	require.NoError(t, err)
	require.Equal(t, v, c.delay)

	v = 0
	err = WithDelay(v)(c)
	require.Error(t, err)
}

func TestWithDelayFactor(t *testing.T) {
	t.Parallel()

	var v float64

	c := defaultHTTPRetrier()

	v = 1.5
	err := WithDelayFactor(v)(c)
	require.NoError(t, err)
	require.InDelta(t, v, c.delayFactor, 0.001)

	v = 0
	err = WithDelayFactor(v)(c)
	require.Error(t, err)
}

func TestWithJitter(t *testing.T) {
	t.Parallel()

	var v time.Duration

	c := defaultHTTPRetrier()

	v = 131 * time.Millisecond
	err := WithJitter(v)(c)
	require.NoError(t, err)
	require.Equal(t, v, c.jitter)

	v = 0
	err = WithJitter(v)(c)
	require.Error(t, err)
}
