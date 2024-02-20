package periodic

import (
	"context"
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
		tt := tt
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

	ctx := context.TODO()

	p.Start(ctx)

	wait := 3 * interval
	time.Sleep(wait)

	d := <-p.resetTimer
	require.GreaterOrEqual(t, wait, d)

	require.NoError(t, ctx.Err())

	p.Stop()

	require.LessOrEqual(t, 2, <-count)
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
