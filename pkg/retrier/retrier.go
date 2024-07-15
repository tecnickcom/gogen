/*
Package retrier provides the ability to automatically repeat a user-defined
function based on the error status.

The default behavior is to retry in case of any error.

This package provides a ready-made DefaultRetryIf function that can be used for
most cases. Additionally, it allows to set the maximum number of retries, the
delay after the first failed attempt, the time multiplication factor to
determine the successive delay value, and the jitter used to introduce
randomness and avoid request collisions.
*/
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
	taskError         error
}

// defaultRetrier returns a new instance of Retrier with default configuration values.
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

	rctx, cancel := context.WithCancel(ctx)
	defer cancel()

	r.timer = time.NewTimer(1 * time.Nanosecond)

	for {
		select {
		case <-rctx.Done():
			return fmt.Errorf("main context has been canceled: %w", rctx.Err())
		case d := <-r.resetTimer:
			r.setTimer(d)
		case <-r.timer.C:
			if r.exec(rctx, task) {
				return r.taskError
			}
		}
	}
}

// setTimer sets the timer for the Retrier with the given duration.
// If the timer is already running, it is stopped and the timer channel is drained before resetting.
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

// exec executes the given task function with a timeout and handles retries if necessary.
// It returns true if the task should not be retried or if the maximum number of attempts has been reached.
// Otherwise, it returns false to indicate that the task should be retried.
func (r *Retrier) exec(ctx context.Context, task TaskFn) bool {
	tctx, cancel := context.WithTimeout(ctx, r.timeout)
	r.taskError = task(tctx)

	cancel()

	r.remainingAttempts--
	if r.remainingAttempts == 0 || !r.retryIfFn(r.taskError) {
		return true
	}

	r.resetTimer <- time.Duration(int64(r.nextDelay) + rand.Int63n(int64(r.jitter))) //nolint:gosec
	r.nextDelay *= r.delayFactor

	return false
}
