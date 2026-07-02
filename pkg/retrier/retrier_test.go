package retrier

import (
	"context"
	"errors"
	"math"
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

// TestRun_exec_backoff_overflow_clamp verifies the exponential backoff delay
// stays positive (no float64 to int64 conversion overflow) at high attempt
// counts.
func TestRun_exec_backoff_overflow_clamp(t *testing.T) {
	t.Parallel()

	r, err := New(
		WithAttempts(100),
		WithDelay(1*time.Second),
		WithDelayFactor(2),
		WithJitter(1*time.Millisecond),
		WithTimeout(1*time.Second),
	)
	require.NoError(t, err)

	s := &run{
		cfg:               r,
		timer:             time.NewTimer(1 * time.Hour),
		nextDelay:         float64(r.delay),
		remainingAttempts: r.attempts,
	}
	defer s.timer.Stop()

	task := func(_ context.Context) error { return errTask }

	// Drive exec through enough failed attempts that the uncapped delay
	// (1s doubled on each retry) would overflow int64 after ~33 doublings.
	for range 99 {
		stop, execErr := s.exec(t.Context(), task)
		require.False(t, stop)
		require.ErrorIs(t, execErr, errTask)

		require.Positive(t, s.nextDelay)
		require.LessOrEqual(t, s.nextDelay, maxBackoffDelay)
		require.Positive(t, time.Duration(int64(s.nextDelay)),
			"backoff delay must stay positive at high attempt counts")
	}
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

func TestRun_setTimer(t *testing.T) {
	t.Parallel()

	s := &run{
		timer: time.NewTimer(1 * time.Millisecond),
	}

	time.Sleep(2 * time.Millisecond)
	s.setTimer(2 * time.Millisecond)

	<-s.timer.C
}

func Test_setTimer_drainsFiredTimer(t *testing.T) {
	t.Parallel()

	timer := time.NewTimer(time.Nanosecond)

	time.Sleep(10 * time.Millisecond) // let the timer fire so Stop returns false

	s := &run{timer: timer}
	s.setTimer(time.Hour) // must drain the fired value before resetting

	require.True(t, s.timer.Stop())
}

func Test_exec_delayOverflowClamp(t *testing.T) {
	t.Parallel()

	cfg := defaultRetrier()
	cfg.jitter = time.Duration(math.MaxInt64)
	cfg.retryIfFn = func(error) bool { return true }

	s := &run{
		cfg:   cfg,
		timer: time.NewTimer(time.Hour),
	}
	defer s.timer.Stop()

	task := func(_ context.Context) error { return errTask }

	// delay = maxBackoffDelay + rand[0, MaxInt64) wraps negative with
	// probability ~1/2 per attempt, so 100 attempts make missing the
	// overflow-clamp branch vanishingly unlikely (~2^-100).
	for range 100 {
		s.nextDelay = maxBackoffDelay
		s.remainingAttempts = 2

		stop, err := s.exec(t.Context(), task)
		require.False(t, stop)
		require.ErrorIs(t, err, errTask)
	}
}
