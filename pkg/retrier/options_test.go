package retrier

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWithRetryIfFn(t *testing.T) {
	t.Parallel()

	r := &Retrier{}

	v := func(err error) bool { return true }
	err := WithRetryIfFn(v)(r)
	require.NoError(t, err)

	v = nil
	err = WithRetryIfFn(v)(r)
	require.Error(t, err)
}

func TestWithAttempts(t *testing.T) {
	t.Parallel()

	var v uint

	r := defaultRetrier()

	v = 5
	err := WithAttempts(v)(r)
	require.NoError(t, err)
	require.Equal(t, v, r.attempts)

	v = 0
	err = WithAttempts(v)(r)
	require.Error(t, err)
}

func TestWithDelay(t *testing.T) {
	t.Parallel()

	var v time.Duration

	r := defaultRetrier()

	v = 503 * time.Millisecond
	err := WithDelay(v)(r)
	require.NoError(t, err)
	require.Equal(t, v, r.delay)

	v = 0
	err = WithDelay(v)(r)
	require.Error(t, err)
}

func TestWithDelayFactor(t *testing.T) {
	t.Parallel()

	var v float64

	r := defaultRetrier()

	v = 1.5
	err := WithDelayFactor(v)(r)
	require.NoError(t, err)
	require.Equal(t, v, r.delayFactor)

	v = 0
	err = WithDelayFactor(v)(r)
	require.Error(t, err)
}

func TestWithJitter(t *testing.T) {
	t.Parallel()

	var v time.Duration

	r := defaultRetrier()

	v = 131 * time.Millisecond
	err := WithJitter(v)(r)
	require.NoError(t, err)
	require.Equal(t, v, r.jitter)

	v = 0
	err = WithJitter(v)(r)
	require.Error(t, err)
}

func TestWithTimeout(t *testing.T) {
	t.Parallel()

	var v time.Duration

	r := defaultRetrier()

	v = 283 * time.Millisecond
	err := WithTimeout(v)(r)
	require.NoError(t, err)
	require.Equal(t, v, r.timeout)

	v = 0
	err = WithTimeout(v)(r)
	require.Error(t, err)
}
