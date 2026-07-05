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

// WithAttempts sets the maximum number of total attempts — the initial request
// plus retries (default 4); must be at least 1.
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

// WithMaxDelay sets the upper bound for the computed exponential backoff delay
// (before jitter), preventing unbounded growth and overflow (default 30s).
func WithMaxDelay(maxDelay time.Duration) Option {
	return func(r *HTTPRetrier) error {
		if int64(maxDelay) < 1 {
			return errors.New("max delay must be greater than zero")
		}

		r.maxDelay = maxDelay

		return nil
	}
}

// WithJitterStrategy selects how backoff jitter is applied (default [JitterAdditive]).
// [JitterFull] and [JitterEqual] scale jitter with the delay for better decorrelation.
// Returns error if the strategy is not a defined [JitterStrategy].
func WithJitterStrategy(strategy JitterStrategy) Option {
	return func(r *HTTPRetrier) error {
		if !strategy.Valid() {
			return errors.New("invalid jitter strategy")
		}

		r.strategy = strategy

		return nil
	}
}

// WithRespectRetryAfter makes the retrier honor a response's Retry-After header
// (delta-seconds or HTTP-date), waiting at least that long (plus jitter) before
// the next attempt, capped at [WithMaxRetryAfter] (default [DefaultMaxRetryAfter]).
// This can exceed [WithMaxDelay], which bounds only the exponential backoff; a
// request context deadline is the ultimate bound on any wait. Default: disabled.
func WithRespectRetryAfter() Option {
	return func(r *HTTPRetrier) error {
		r.respectRetryAfter = true

		return nil
	}
}

// WithMaxRetryAfter caps the delay honored from a Retry-After header
// (default [DefaultMaxRetryAfter], 24h). Only relevant with [WithRespectRetryAfter].
// Returns error if maxRetryAfter < 1 nanosecond.
func WithMaxRetryAfter(maxRetryAfter time.Duration) Option {
	return func(r *HTTPRetrier) error {
		if int64(maxRetryAfter) < 1 {
			return errors.New("max retry-after must be greater than zero")
		}

		r.maxRetryAfter = maxRetryAfter

		return nil
	}
}

// WithOnRetry registers an observability callback invoked before each scheduled
// retry (see [OnRetryFn]). Returns error if the callback is nil.
func WithOnRetry(onRetry OnRetryFn) Option {
	return func(r *HTTPRetrier) error {
		if onRetry == nil {
			return errors.New("the onRetry callback is required")
		}

		r.onRetry = onRetry

		return nil
	}
}
