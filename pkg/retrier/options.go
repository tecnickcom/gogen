package retrier

import (
	"errors"
	"time"
)

// Option is the interface that allows to set the options.
type Option func(c *Retrier) error

// WithRetryIfFn customizes the retry condition predicate.
// Returns error if the function is nil.
func WithRetryIfFn(retryIfFn RetryIfFn) Option {
	return func(r *Retrier) error {
		if retryIfFn == nil {
			return errors.New("the retry function is required")
		}

		r.retryIfFn = retryIfFn

		return nil
	}
}

// WithAttempts customizes the maximum number of retry attempts.
// Returns error if attempts < 1.
func WithAttempts(attempts uint) Option {
	return func(r *Retrier) error {
		if attempts < 1 {
			return errors.New("the number of attempts must be at least 1")
		}

		r.attempts = attempts

		return nil
	}
}

// WithDelay customizes the base delay after the first failed attempt.
// Returns error if delay < 1 nanosecond.
func WithDelay(delay time.Duration) Option {
	return func(r *Retrier) error {
		if int64(delay) < 1 {
			return errors.New("delay must be greater than zero")
		}

		r.delay = delay

		return nil
	}
}

// WithDelayFactor customizes the exponential backoff multiplier (factor > 1 for exponential growth).
// Returns error if delayFactor < 1.
func WithDelayFactor(delayFactor float64) Option {
	return func(r *Retrier) error {
		if delayFactor < 1 {
			return errors.New("delay factor must be at least 1")
		}

		r.delayFactor = delayFactor

		return nil
	}
}

// WithJitter customizes the maximum random jitter added to each retry delay to avoid thundering-herd.
// Returns error if jitter < 1 nanosecond.
func WithJitter(jitter time.Duration) Option {
	return func(r *Retrier) error {
		if int64(jitter) < 1 {
			return errors.New("jitter must be greater than zero")
		}

		r.jitter = jitter

		return nil
	}
}

// WithTimeout customizes the per-attempt timeout applied via context.WithTimeout().
// Returns error if timeout < 1 nanosecond.
func WithTimeout(timeout time.Duration) Option {
	return func(r *Retrier) error {
		if int64(timeout) < 1 {
			return errors.New("timeout must be greater than zero")
		}

		r.timeout = timeout

		return nil
	}
}
