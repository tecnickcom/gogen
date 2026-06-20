package periodic

import (
	"context"
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

	wait := 3 * interval
	time.Sleep(wait)

	d := <-p.resetTimer
	require.GreaterOrEqual(t, wait, d)

	require.NoError(t, ctx.Err())

	p.Stop()

	require.LessOrEqual(t, 2, <-count)
}

func Test_Stop_before_start(t *testing.T) {
	t.Parallel()

	p, err := New(10*time.Millisecond, 0, time.Millisecond, func(_ context.Context) {})
	require.NoError(t, err)

	p.Stop() // must be a no-op (and must not block) when Start was never called
}

func Test_run_zero_jitter(t *testing.T) {
	t.Parallel()

	p := &Periodic{
		interval:   int64(10 * time.Millisecond),
		jitter:     0,
		timeout:    1 * time.Millisecond,
		task:       func(_ context.Context) {},
		resetTimer: make(chan time.Duration, 1),
	}

	// Must not panic on rand.Int63n(0) and must reset to exactly the interval.
	p.run(t.Context())

	require.Equal(t, 10*time.Millisecond, <-p.resetTimer)
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

func TestPeriodic_setTimer(t *testing.T) {
	t.Parallel()

	c := &Periodic{
		timer: time.NewTimer(1 * time.Millisecond),
	}

	time.Sleep(10 * time.Millisecond)
	c.setTimer(2 * time.Millisecond)
	<-c.timer.C
}
