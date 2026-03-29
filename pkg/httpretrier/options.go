package httpretrier

import (
	"errors"
	"time"
)

// Option configures an [HTTPRetrier] instance.
type Option func(c *HTTPRetrier) error

// WithRetryIfFn customizes retry decision function (default: retry only on transport errors).
func WithRetryIfFn(retryIfFn RetryIfFn) Option {
	return func(r *HTTPRetrier) error {
		if retryIfFn == nil {
			return errors.New("the retry function is required")
		}

		r.retryIfFn = retryIfFn

		return nil
	}
}

// WithAttempts sets maximum retry attempts (default 4), must be at least 1.
func WithAttempts(attempts uint) Option {
	return func(r *HTTPRetrier) error {
		if attempts < 1 {
			return errors.New("the number of attempts must be at least 1")
		}

		r.attempts = attempts

		return nil
	}
}

// WithDelay sets initial delay between first failed attempt and first retry (default 1 second).
func WithDelay(delay time.Duration) Option {
	return func(r *HTTPRetrier) error {
		if int64(delay) < 1 {
			return errors.New("delay must be greater than zero")
		}

		r.delay = delay

		return nil
	}
}

// WithDelayFactor sets exponential backoff multiplier (default 2): next_delay = current_delay * delayFactor.
func WithDelayFactor(delayFactor float64) Option {
	return func(r *HTTPRetrier) error {
		if delayFactor < 1 {
			return errors.New("delay factor must be at least 1")
		}

		r.delayFactor = delayFactor

		return nil
	}
}

// WithJitter sets random jitter ceiling to prevent synchronized retry storms (default 100ms).
func WithJitter(jitter time.Duration) Option {
	return func(r *HTTPRetrier) error {
		if int64(jitter) < 1 {
			return errors.New("jitter must be greater than zero")
		}

		r.jitter = jitter

		return nil
	}
}
