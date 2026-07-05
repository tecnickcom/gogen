package retrier

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errTask is a reusable sentinel error for tests.
var errTask = errors.New("ERROR")

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "succeeds with defaults",
			wantErr: false,
		},
		{
			name: "succeeds with custom values",
			opts: []Option{
				WithRetryIfFn(func(_ error) bool { return true }),
				WithAttempts(5),
				WithDelay(601 * time.Millisecond),
				WithDelayFactor(1.3),
				WithJitter(109 * time.Millisecond),
				WithTimeout(131 * time.Millisecond),
				WithMaxDelay(7 * time.Second),
				WithJitterStrategy(JitterFull),
				WithOnRetry(func(_ uint, _ time.Duration, _ error) {}),
			},
			wantErr: false,
		},
		{
			name:    "fails with invalid option",
			opts:    []Option{WithJitter(0)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, err := New(tt.opts...)

			if tt.wantErr {
				require.Nil(t, r)
				require.Error(t, err, "New() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			require.NotNil(t, r, "New() returned value should not be nil")
			require.NoError(t, err)
		})
	}
}

func TestRetrier_Run(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		newTask func() TaskFn
		timeout time.Duration
		wantErr bool
	}{
		{
			name:    "success at first attempt",
			newTask: func() TaskFn { return func(_ context.Context) error { return nil } },
			timeout: 1 * time.Second,
		},
		{
			name: "success at third attempt",
			newTask: func() TaskFn {
				var count int

				return func(_ context.Context) error {
					if count == 2 {
						return nil
					}

					count++

					return errTask
				}
			},
			timeout: 1 * time.Second,
		},
		{
			name:    "fail all attempts",
			newTask: func() TaskFn { return func(_ context.Context) error { return errTask } },
			timeout: 1 * time.Second,
			wantErr: true,
		},
		{
			name:    "fail with main timeout",
			newTask: func() TaskFn { return func(ctx context.Context) error { <-ctx.Done(); return errTask } },
			timeout: 1 * time.Millisecond,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := []Option{
				WithRetryIfFn(DefaultRetryIf),
				WithAttempts(4),
				WithDelay(10 * time.Millisecond),
				WithDelayFactor(1.1),
				WithJitter(5 * time.Millisecond),
				WithTimeout(2 * time.Millisecond),
			}

			r, err := New(opts...)
			require.NoError(t, err)
			require.NotNil(t, r)

			ctx, cancel := context.WithTimeout(t.Context(), tt.timeout)
			defer cancel()

			err = r.Run(ctx, tt.newTask())
			require.Equal(t, tt.wantErr, err != nil, "Run() error = %v, wantErr %v", err, tt.wantErr)
		})
	}
}

// TestRetrier_Run_concurrent verifies that a single configured Retrier is safe
// to share and to Run concurrently from many goroutines (must pass -race).
func TestRetrier_Run_concurrent(t *testing.T) {
	t.Parallel()

	r, err := New(
		WithAttempts(3),
		WithDelay(1*time.Millisecond),
		WithDelayFactor(2),
		WithJitter(1*time.Millisecond),
		WithTimeout(20*time.Millisecond),
	)
	require.NoError(t, err)

	const goroutines = 32

	var (
		wg        sync.WaitGroup
		successes atomic.Int64
		failures  atomic.Int64
	)

	for i := range goroutines {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			var calls int

			// Each goroutine keeps its own counter so attempts never share state.
			task := func(_ context.Context) error {
				calls++

				// Half the goroutines always fail to exercise the retry path.
				if id%2 == 0 {
					return errTask
				}

				return nil
			}

			runErr := r.Run(t.Context(), task)

			if id%2 == 0 {
				assert.Error(t, runErr)
				assert.Equal(t, 3, calls, "even goroutine should exhaust all attempts")
				failures.Add(1)
			} else {
				assert.NoError(t, runErr)
				assert.Equal(t, 1, calls, "odd goroutine should succeed on first attempt")
				successes.Add(1)
			}
		}(i)
	}

	wg.Wait()

	require.Equal(t, int64(goroutines/2), successes.Load())
	require.Equal(t, int64(goroutines/2), failures.Load())
}

// TestRetrier_Run_backoffGrowth verifies the delay grows by the configured
// factor across successive attempts.
func TestRetrier_Run_backoffGrowth(t *testing.T) {
	t.Parallel()

	r, err := New(
		WithAttempts(4),
		WithDelay(20*time.Millisecond),
		WithDelayFactor(2),
		WithJitter(1*time.Nanosecond),
		WithTimeout(1*time.Second),
	)
	require.NoError(t, err)

	var timestamps []time.Time

	task := func(_ context.Context) error {
		timestamps = append(timestamps, time.Now())

		return errTask
	}

	runErr := r.Run(t.Context(), task)
	require.Error(t, runErr)
	require.Len(t, timestamps, 4, "all attempts should run")

	// gaps between attempts should roughly double: ~20ms, ~40ms, ~80ms.
	gap1 := timestamps[1].Sub(timestamps[0])
	gap2 := timestamps[2].Sub(timestamps[1])
	gap3 := timestamps[3].Sub(timestamps[2])

	require.Greater(t, gap2, gap1, "second delay should exceed the first")
	require.Greater(t, gap3, gap2, "third delay should exceed the second")
}

// TestRetrier_Run_maxDelayCaps verifies WithMaxDelay is wired into the backoff
// so the gap between attempts stops growing once the cap is reached (rather than
// following the raw exponential progression).
func TestRetrier_Run_maxDelayCaps(t *testing.T) {
	t.Parallel()

	const maxDelay = 30 * time.Millisecond

	r, err := New(
		WithAttempts(4),
		WithDelay(10*time.Millisecond),
		WithDelayFactor(8),            // uncapped 3rd gap would be ~640ms
		WithJitter(1*time.Nanosecond), // 1ns ceiling -> jitter is always 0
		WithMaxDelay(maxDelay),
		WithTimeout(1*time.Second),
	)
	require.NoError(t, err)

	var timestamps []time.Time

	task := func(_ context.Context) error {
		timestamps = append(timestamps, time.Now())

		return errTask
	}

	runErr := r.Run(t.Context(), task)
	require.Error(t, runErr)
	require.Len(t, timestamps, 4, "all attempts should run")

	// Without the cap the third gap would be ~640ms; capped it sits near 30ms.
	// A generous 4x ceiling distinguishes capped from uncapped without being
	// flaky under scheduler jitter.
	gap3 := timestamps[3].Sub(timestamps[2])
	require.Greater(t, gap3, maxDelay/2, "capped gap should still be near the cap")
	require.Less(t, gap3, 4*maxDelay, "gap must be capped well below the raw exponential value")
}

// TestRetrier_Run_onRetry verifies the observability callback fires once per
// scheduled retry with increasing 1-based attempt numbers.
func TestRetrier_Run_onRetry(t *testing.T) {
	t.Parallel()

	var attempts []uint

	onRetry := func(attempt uint, _ time.Duration, err error) {
		attempts = append(attempts, attempt)

		assert.ErrorIs(t, err, errTask)
	}

	r, err := New(
		WithAttempts(3),
		WithDelay(1*time.Millisecond),
		WithJitter(1*time.Millisecond),
		WithTimeout(time.Second),
		WithOnRetry(onRetry),
	)
	require.NoError(t, err)

	runErr := r.Run(t.Context(), func(_ context.Context) error { return errTask })
	require.Error(t, runErr)

	// 3 attempts -> 2 retries scheduled, so onRetry fires for attempts 1 and 2
	// (the final, exhausted attempt does not schedule a retry).
	require.Equal(t, []uint{1, 2}, attempts)
}

// TestRetrier_Run_onRetryNotCalledOnCancel verifies onRetry does not fire for a
// retry preempted by context cancellation during the attempt.
func TestRetrier_Run_onRetryNotCalledOnCancel(t *testing.T) {
	t.Parallel()

	var onRetryCalls int

	onRetry := func(_ uint, _ time.Duration, _ error) { onRetryCalls++ }

	r, err := New(
		WithAttempts(3),
		WithDelay(time.Millisecond),
		WithJitter(time.Millisecond),
		WithTimeout(time.Second),
		WithOnRetry(onRetry),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())

	runErr := r.Run(ctx, func(_ context.Context) error {
		cancel() // context ends during the attempt

		return errTask
	})

	require.Error(t, runErr)
	require.Zero(t, onRetryCalls, "onRetry must not fire for a retry preempted by cancellation")
}

// TestRetrier_Run_preCanceledContext verifies Run fails fast without running the
// task when the context is already done.
func TestRetrier_Run_preCanceledContext(t *testing.T) {
	t.Parallel()

	r, err := New(WithAttempts(3), WithDelay(time.Millisecond), WithJitter(time.Millisecond))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var calls int

	runErr := r.Run(ctx, func(_ context.Context) error {
		calls++

		return nil
	})

	require.Error(t, runErr)
	require.Zero(t, calls, "task must not run when the context is already canceled")
}

// TestRetrier_Run_earlyStop verifies that a RetryIfFn returning false stops
// retrying immediately even when attempts remain.
func TestRetrier_Run_earlyStop(t *testing.T) {
	t.Parallel()

	r, err := New(
		WithRetryIfFn(func(_ error) bool { return false }),
		WithAttempts(5),
		WithDelay(1*time.Millisecond),
		WithDelayFactor(2),
		WithJitter(1*time.Millisecond),
		WithTimeout(1*time.Second),
	)
	require.NoError(t, err)

	var calls int

	task := func(_ context.Context) error {
		calls++

		return errTask
	}

	runErr := r.Run(t.Context(), task)
	require.Error(t, runErr)
	require.Equal(t, 1, calls, "RetryIfFn returning false should stop after the first attempt")
}

func TestDefaultRetryIf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "true with error",
			err:  errors.New("ERROR"),
			want: true,
		},
		{
			name: "false with no error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := DefaultRetryIf(tt.err)

			require.Equal(t, tt.want, got)
		})
	}
}
