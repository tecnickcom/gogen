package httpretrier

import (
	"fmt"
	"time"
)

// Option is the interface that allows to set the options.
type Option func(c *HTTPRetrier) error

// WithRetryIfFn set the function used to decide when retry.
func WithRetryIfFn(retryIfFn RetryIfFn) Option {
	return func(r *HTTPRetrier) error {
		if retryIfFn == nil {
			return fmt.Errorf("the retry function is required")
		}

		r.retryIfFn = retryIfFn

		return nil
	}
}

// WithAttempts set the maximum number of retries.
func WithAttempts(attempts uint) Option {
	return func(r *HTTPRetrier) error {
		if attempts < 1 {
			return fmt.Errorf("the number of attempts must be at least 1")
		}

		r.attempts = attempts

		return nil
	}
}

// WithDelay set the delay after the first failed attempt.
func WithDelay(delay time.Duration) Option {
	return func(r *HTTPRetrier) error {
		if int64(delay) < 1 {
			return fmt.Errorf("delay must be greater than zero")
		}

		r.delay = delay

		return nil
	}
}

// WithDelayFactor set the multiplication factor to get the successive delay value.
// A delay factor greater than 1 means an exponential delay increase.
// if the delay factor is 2 and the first delay is 1, then the delays will be: [1, 2, 4, 8, ...].
func WithDelayFactor(delayFactor float64) Option {
	return func(r *HTTPRetrier) error {
		if delayFactor < 1 {
			return fmt.Errorf("delay factor must be at least 1")
		}

		r.delayFactor = delayFactor

		return nil
	}
}

// WithJitter sets the maximum random Jitter time between retries.
// This is useful to avoid the Thundering herd problem (https://en.wikipedia.org/wiki/Thundering_herd_problem).
func WithJitter(jitter time.Duration) Option {
	return func(r *HTTPRetrier) error {
		if int64(jitter) < 1 {
			return fmt.Errorf("jitter must be greater than zero")
		}

		r.jitter = jitter

		return nil
	}
}
