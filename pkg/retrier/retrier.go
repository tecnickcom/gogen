/*
Package retrier provides a configurable retry engine for executing a task
function with backoff, jitter, and per-attempt timeouts.

# Problem

Transient failures are common in distributed systems (temporary network issues,
short-lived upstream overload, brief lock/contention windows). Retrying can
dramatically improve success rates, but ad hoc retry loops often miss important
details such as cancellation propagation, bounded attempts, timeout isolation
per attempt, and jitter to avoid synchronized retry storms.

This package centralizes those concerns in a reusable retrier implementation.

# How It Works

[New] creates a [Retrier] with defaults, or with custom [Option] values.
[Retrier.Run] then executes a [TaskFn] according to configured retry rules:

 1. Execute the task with a per-attempt timeout context.
 2. Evaluate the result with a retry predicate ([RetryIfFn]).
 3. Stop when attempts are exhausted or retry is not required.
 4. Otherwise schedule the next attempt after delay + random jitter.
 5. Increase delay by the configured multiplication factor for successive
    retries.

The run loop always respects parent context cancellation.

# Defaults

  - attempts: [DefaultAttempts] (4)
  - initial delay: [DefaultDelay] (1s)
  - delay factor: [DefaultDelayFactor] (2)
  - jitter: [DefaultJitter] (1ms)
  - per-attempt timeout: [DefaultTimeout] (1s)
  - retry condition: [DefaultRetryIf] (retry on any non-nil error)

# Key Features

  - Pluggable retry condition via [WithRetryIfFn].
  - Bounded retry count via [WithAttempts].
  - Configurable delay, exponential factor, and jitter via [WithDelay],
    [WithDelayFactor], and [WithJitter].
  - Per-attempt timeout isolation via [WithTimeout].
  - Context-aware cancellation for clean shutdown behavior.

# Usage

	r, err := retrier.New(
	    retrier.WithAttempts(5),
	    retrier.WithDelay(200*time.Millisecond),
	    retrier.WithDelayFactor(2),
	    retrier.WithJitter(25*time.Millisecond),
	)
	if err != nil {
	    return err
	}

	err = r.Run(ctx, func(ctx context.Context) error {
	    return callExternalService(ctx)
	})
	if err != nil {
	    return err
	}

This package is ideal for retrying idempotent or safe-to-repeat operations in
networked and high-concurrency Go services.
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

// Retrier applies configurable retry logic to generic task functions.
//
// A configured Retrier is immutable: it holds no per-run state, so a single
// instance is safe to share and to call concurrently from multiple goroutines.
type Retrier struct {
	delayFactor float64
	attempts    uint
	delay       time.Duration
	jitter      time.Duration
	timeout     time.Duration
	retryIfFn   RetryIfFn
}

// defaultRetrier returns a [Retrier] initialized with package defaults.
func defaultRetrier() *Retrier {
	return &Retrier{
		attempts:    DefaultAttempts,
		delay:       DefaultDelay,
		delayFactor: DefaultDelayFactor,
		jitter:      DefaultJitter,
		timeout:     DefaultTimeout,
		retryIfFn:   DefaultRetryIf,
	}
}

// DefaultRetryIf is the default retry predicate: returns true if error is non-nil.
func DefaultRetryIf(err error) bool {
	return err != nil
}

// New constructs a Retrier with defaults, applying optional configuration.
func New(opts ...Option) (*Retrier, error) {
	r := defaultRetrier()

	for _, applyOpt := range opts {
		err := applyOpt(r)
		if err != nil {
			return nil, err
		}
	}

	return r, nil
}

// run holds the mutable per-call state for a single [Retrier.Run] invocation.
//
// Keeping this state local (instead of on the shared [Retrier]) makes the
// configured [Retrier] immutable and safe to call concurrently.
type run struct {
	cfg               *Retrier
	timer             *time.Timer
	taskError         error
	nextDelay         float64
	remainingAttempts uint
}

// Run executes the task with exponential backoff and jitter, respecting parent context cancellation.
func (r *Retrier) Run(ctx context.Context, task TaskFn) error {
	rctx, cancel := context.WithCancel(ctx)
	defer cancel()

	st := &run{
		cfg:               r,
		timer:             time.NewTimer(1 * time.Nanosecond),
		nextDelay:         float64(r.delay),
		remainingAttempts: r.attempts,
	}
	defer st.timer.Stop()

	for {
		select {
		case <-rctx.Done():
			return fmt.Errorf("main context has been canceled: %w", rctx.Err())
		case <-st.timer.C:
			stop, err := st.exec(rctx, task)
			if stop {
				return err
			}
		}
	}
}

// setTimer resets the per-run timer, draining its channel if necessary to prevent deadlocks.
func (s *run) setTimer(d time.Duration) {
	if !s.timer.Stop() {
		// make sure to drain timer channel before reset
		select {
		case <-s.timer.C:
		default:
		}
	}

	s.timer.Reset(d)
}

// exec runs the task with a per-attempt timeout, evaluates the retry predicate, and schedules the next attempt if needed.
// It returns stop=true to end retrying (success, exhausted attempts, or retry not needed) along with the last task error.
func (s *run) exec(ctx context.Context, task TaskFn) (bool, error) {
	tctx, cancel := context.WithTimeout(ctx, s.cfg.timeout)
	s.taskError = task(tctx)

	cancel()

	s.remainingAttempts--
	if s.remainingAttempts == 0 || !s.cfg.retryIfFn(s.taskError) {
		return true, s.taskError
	}

	s.setTimer(time.Duration(int64(s.nextDelay) + rand.Int63n(int64(s.cfg.jitter)))) //nolint:gosec

	s.nextDelay *= s.cfg.delayFactor

	return false, s.taskError
}
