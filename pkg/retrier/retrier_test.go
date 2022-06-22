package retrier

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
				WithRetryIfFn(func(err error) bool { return true }),
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
		tt := tt
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

	var count int

	tests := []struct {
		name                  string
		task                  TaskFn
		timeout               time.Duration
		wantRemainingAttempts uint
		wantErr               bool
	}{
		{
			name:                  "success at first attempt",
			task:                  func(_ context.Context) error { return nil },
			timeout:               1 * time.Second,
			wantRemainingAttempts: 3,
		},
		{
			name: "success at third attempt",
			task: func(_ context.Context) error {
				if count == 2 {
					return nil
				}
				count++
				return fmt.Errorf("ERROR")
			},
			timeout:               1 * time.Second,
			wantRemainingAttempts: 1,
		},
		{
			name:                  "fail all attempts",
			task:                  func(_ context.Context) error { return fmt.Errorf("ERROR") },
			timeout:               1 * time.Second,
			wantRemainingAttempts: 0,
			wantErr:               true,
		},
		{
			name:                  "fail with main timeout",
			task:                  func(ctx context.Context) error { <-ctx.Done(); return fmt.Errorf("ERROR") },
			timeout:               1 * time.Millisecond,
			wantRemainingAttempts: 3,
			wantErr:               true,
		},
	}

	for _, tt := range tests {
		tt := tt
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

			ctx, cancel := context.WithTimeout(context.TODO(), tt.timeout)
			defer cancel()

			err = r.Run(ctx, tt.task)
			require.Equal(t, tt.wantErr, err != nil, "Run() error = %v, wantErr %v", err, tt.wantErr)
			require.Equal(t, tt.wantRemainingAttempts, r.remainingAttempts, "Run() remainingAttempts = %v, wantRemainingAttempts %v", err, tt.wantErr)
		})
	}
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
			err:  fmt.Errorf("ERROR"),
			want: true,
		},
		{
			name: "false with no error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := DefaultRetryIf(tt.err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestRetrier_setTimer(t *testing.T) {
	t.Parallel()

	r := &Retrier{
		timer: time.NewTimer(1 * time.Millisecond),
	}

	time.Sleep(2 * time.Millisecond)
	r.setTimer(2 * time.Millisecond)
	<-r.timer.C
}
