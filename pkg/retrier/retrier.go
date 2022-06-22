// Package retrier allow to retry execute a function in case of errors.
package retrier

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

const (
	// DefaultAttempts is the default maximum number of retry attempts.
	DefaultAttempts = 4

	// DefaultDelay is the default delay to apply after the first failed attempt.
	DefaultDelay = 1 * time.Second

	// DefaultDelayFactor is the default multiplication factor to get the successive delay value.
	DefaultDelayFactor = 2

	// DefaultJitter is the default maximum random Jitter time between retries.
	DefaultJitter = 1 * time.Millisecond

	// DefaultTimeout is the default timeout applied to each function call via context.
	DefaultTimeout = 1 * time.Second
)

// TaskFn is the type of function to be executed.
type TaskFn func(ctx context.Context) error

// RetryIfFn is the signature of the function used to decide when retry.
type RetryIfFn func(err error) bool

// Retrier represents an instance of the HTTP retrier.
type Retrier struct {
	nextDelay         float64
	delayFactor       float64
	attempts          uint
	remainingAttempts uint
	delay             time.Duration
	jitter            time.Duration
	timeout           time.Duration
	retryIfFn         RetryIfFn
	timer             *time.Timer
	resetTimer        chan time.Duration
	ctx               context.Context
	cancel            context.CancelFunc
	taskError         error
}

func defaultRetrier() *Retrier {
	return &Retrier{
		attempts:    DefaultAttempts,
		delay:       DefaultDelay,
		delayFactor: DefaultDelayFactor,
		jitter:      DefaultJitter,
		timeout:     DefaultTimeout,
		retryIfFn:   DefaultRetryIf,
		resetTimer:  make(chan time.Duration, 1),
	}
}

// DefaultRetryIf is the default function to check the retry condition.
func DefaultRetryIf(err error) bool {
	return err != nil
}

// New creates a new instance.
func New(opts ...Option) (*Retrier, error) {
	r := defaultRetrier()

	for _, applyOpt := range opts {
		if err := applyOpt(r); err != nil {
			return nil, err
		}
	}

	return r, nil
}

// Run attempts to execute the task according to the retry rules.
func (r *Retrier) Run(ctx context.Context, task TaskFn) error {
	r.nextDelay = float64(r.delay)
	r.remainingAttempts = r.attempts

	r.ctx, r.cancel = context.WithCancel(ctx)
	defer r.cancel()

	r.timer = time.NewTimer(1 * time.Nanosecond)

	for {
		select {
		case <-r.ctx.Done():
			return fmt.Errorf("main context has been canceled: %w", r.ctx.Err())
		case d := <-r.resetTimer:
			r.setTimer(d)
		case <-r.timer.C:
			if r.exec(task) {
				return r.taskError
			}
		}
	}
}

func (r *Retrier) setTimer(d time.Duration) {
	if !r.timer.Stop() {
		// make sure to drain timer channel before reset
		select {
		case <-r.timer.C:
		default:
		}
	}

	r.timer.Reset(d)
}

func (r *Retrier) exec(task TaskFn) bool {
	ctx, cancel := context.WithTimeout(r.ctx, r.timeout)
	r.taskError = task(ctx)

	cancel()

	r.remainingAttempts--
	if r.remainingAttempts == 0 || !r.retryIfFn(r.taskError) {
		return true
	}

	r.resetTimer <- time.Duration(int64(r.nextDelay) + rand.Int63n(int64(r.jitter))) // nolint:gosec
	r.nextDelay *= r.delayFactor

	return false
}
