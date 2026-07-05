package retrier

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWithRetryIfFn(t *testing.T) {
	t.Parallel()

	r := &Retrier{}

	v := func(_ error) bool { return true }
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
	require.InDelta(t, v, r.delayFactor, 0.001)

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

func TestWithMaxDelay(t *testing.T) {
	t.Parallel()

	var v time.Duration

	r := defaultRetrier()

	v = 17 * time.Second
	err := WithMaxDelay(v)(r)
	require.NoError(t, err)
	require.Equal(t, v, r.maxDelay)

	v = 0
	err = WithMaxDelay(v)(r)
	require.Error(t, err)
}

func TestWithJitterStrategy(t *testing.T) {
	t.Parallel()

	r := defaultRetrier()

	err := WithJitterStrategy(JitterFull)(r)
	require.NoError(t, err)
	require.Equal(t, JitterFull, r.strategy)

	err = WithJitterStrategy(JitterStrategy(99))(r)
	require.Error(t, err)
}

func TestWithOnRetry(t *testing.T) {
	t.Parallel()

	r := defaultRetrier()

	err := WithOnRetry(func(_ uint, _ time.Duration, _ error) {})(r)
	require.NoError(t, err)
	require.NotNil(t, r.onRetry)

	err = WithOnRetry(nil)(r)
	require.Error(t, err)
}
