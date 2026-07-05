package periodic

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		interval time.Duration
		jitter   time.Duration
		timeout  time.Duration
		task     TaskFn
		wantErr  bool
	}{
		{
			name:     "zero interval",
			interval: 0 * time.Millisecond,
			jitter:   3 * time.Millisecond,
			timeout:  10 * time.Millisecond,
			task:     func(_ context.Context) {},
			wantErr:  true,
		},
		{
			name:     "negative interval",
			interval: -30 * time.Millisecond,
			jitter:   3 * time.Millisecond,
			timeout:  10 * time.Millisecond,
			task:     func(_ context.Context) {},
			wantErr:  true,
		},
		{
			name:     "negative jitter",
			interval: 30 * time.Millisecond,
			jitter:   -3 * time.Millisecond,
			timeout:  10 * time.Millisecond,
			task:     func(_ context.Context) {},
			wantErr:  true,
		},
		{
			name:     "zero timeout",
			interval: 30 * time.Millisecond,
			jitter:   3 * time.Millisecond,
			timeout:  0 * time.Millisecond,
			task:     func(_ context.Context) {},
			wantErr:  true,
		},
		{
			name:     "negative timeout",
			interval: 30 * time.Millisecond,
			jitter:   3 * time.Millisecond,
			timeout:  -10 * time.Millisecond,
			task:     func(_ context.Context) {},
			wantErr:  true,
		},
		{
			name:     "nil task",
			interval: 30 * time.Millisecond,
			jitter:   3 * time.Millisecond,
			timeout:  10 * time.Millisecond,
			task:     nil,
			wantErr:  true,
		},
		{
			name:     "success",
			interval: 30 * time.Millisecond,
			jitter:   3 * time.Millisecond,
			timeout:  10 * time.Millisecond,
			task:     func(_ context.Context) {},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, err := New(tt.interval, tt.jitter, tt.timeout, tt.task)

			if tt.wantErr {
				require.Nil(t, p)
				require.Error(t, err)

				return
			}

			require.NotNil(t, p)
			require.NoError(t, err)
		})
	}
}

func Test_Start_Stop(t *testing.T) {
	t.Parallel()

	count := make(chan int, 1)

	defer close(count)

	count <- 0

	task := func(_ context.Context) {
		v := <-count
		count <- (v + 1)
	}

	interval := 10 * time.Millisecond
	p, err := New(interval, 1*time.Millisecond, 1*time.Millisecond, task)
	require.NotNil(t, p)
	require.NoError(t, err)

	ctx := t.Context()

	p.Start(ctx)

	time.Sleep(3 * interval)

	require.NoError(t, ctx.Err())

	p.Stop()

	require.LessOrEqual(t, 2, <-count)
}

func Test_Start_twice_is_noop(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64

	task := func(_ context.Context) { calls.Add(1) }

	p, err := New(10*time.Millisecond, 1*time.Millisecond, 1*time.Millisecond, task)
	require.NoError(t, err)

	ctx := t.Context()

	p.Start(ctx)
	p.Start(ctx) // second Start must be a no-op: no second goroutine, no data race on the timer

	time.Sleep(25 * time.Millisecond)
	p.Stop()

	require.Positive(t, calls.Load())
}

func Test_Start_after_Stop_is_noop(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64

	task := func(_ context.Context) { calls.Add(1) }

	p, err := New(time.Millisecond, 0, time.Millisecond, task)
	require.NoError(t, err)

	p.Stop()             // stop before start
	p.Start(t.Context()) // must be a no-op: the loop never launches

	time.Sleep(10 * time.Millisecond)
	p.Stop()

	require.Zero(t, calls.Load(), "Start after Stop must not launch the loop")
}

func Test_Start_Stop_concurrent(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// Start and Stop race from separate goroutines; must be free of data races and
	// deadlocks whatever the ordering, and always end stopped (must pass -race).
	for range 20 {
		p, err := New(time.Millisecond, 0, time.Millisecond, func(_ context.Context) {})
		require.NoError(t, err)

		var wg sync.WaitGroup

		wg.Add(2)

		go func() { defer wg.Done(); p.Start(ctx) }()
		go func() { defer wg.Done(); p.Stop() }()

		wg.Wait()

		p.Stop() // idempotent: stops the loop if Start won the race, no-op otherwise
	}
}

func Test_Stop_before_start(t *testing.T) {
	t.Parallel()

	p, err := New(10*time.Millisecond, 0, time.Millisecond, func(_ context.Context) {})
	require.NoError(t, err)

	p.Stop() // must be a no-op (and must not block) when Start was never called
}

func Test_nextDelay_zeroJitter(t *testing.T) {
	t.Parallel()

	p := &Periodic{
		interval: 10 * time.Millisecond,
		jitter:   0,
	}

	// With jitter disabled the pause is exactly the interval (and no rand draw
	// on a zero ceiling, so it can never panic).
	require.Equal(t, 10*time.Millisecond, p.nextDelay())
}

func Test_nextDelay_withJitter(t *testing.T) {
	t.Parallel()

	const (
		interval = 10 * time.Millisecond
		jitter   = 5 * time.Millisecond
	)

	p := &Periodic{interval: interval, jitter: jitter}

	for range 200 {
		d := p.nextDelay()
		require.GreaterOrEqual(t, d, interval)
		require.Less(t, d, interval+jitter)
	}
}

func Test_Stop_waits_for_running_task(t *testing.T) {
	t.Parallel()

	running := make(chan struct{}, 1)

	var completed atomic.Bool

	task := func(_ context.Context) {
		select {
		case running <- struct{}{}:
		default:
		}

		time.Sleep(30 * time.Millisecond)
		completed.Store(true)
	}

	p, err := New(50*time.Millisecond, 1*time.Millisecond, 100*time.Millisecond, task)
	require.NoError(t, err)

	p.Start(t.Context())

	<-running // first invocation is in progress

	p.Stop() // must block until the in-flight task returns

	require.True(t, completed.Load(), "Stop must wait for the in-flight task to finish")

	p.Stop() // safe to call again (returns immediately on the closed done channel)
}
