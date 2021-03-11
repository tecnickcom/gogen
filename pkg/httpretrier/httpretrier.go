// Package httpretrier allow to retry HTTP requests based on HTTP status code boolean conditions.
package httpretrier

import (
	"context"
	"math/rand"
	"net/http"
	"time"

	"github.com/nexmoinc/gosrvlib/pkg/logging"
)

const (
	// DefaultAttempts is the default maximum number of retry attempts.
	DefaultAttempts = 4

	// DefaultDelay is the delay to apply after the first failed attempt.
	DefaultDelay = 1 * time.Second

	// DefaultDelayFactor is the default multiplication factor to get the successive delay value.
	DefaultDelayFactor = 2

	// DefaultJitter is the maximum random Jitter time between retries.
	DefaultJitter = 100 * time.Millisecond
)

// RetryIfFn is the signature of the function used to decide when retry.
type RetryIfFn func(statusCode int, err error) bool

// HTTPClient contains the function to perform the actual HTTP request.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// HTTPRetrier represents an instance of the HTTP retrier.
type HTTPRetrier struct {
	nextDelay         float64
	delayFactor       float64
	delay             time.Duration
	jitter            time.Duration
	attempts          uint
	remainingAttempts uint
	retryIfFn         RetryIfFn
	httpClient        HTTPClient
	timer             *time.Timer
	resetTimer        chan time.Duration
	ctx               context.Context
	cancel            context.CancelFunc
	doResponse        *http.Response
	doError           error
}

func defaultHTTPRetrier() *HTTPRetrier {
	return &HTTPRetrier{
		attempts:    DefaultAttempts,
		delay:       DefaultDelay,
		delayFactor: DefaultDelayFactor,
		jitter:      DefaultJitter,
		retryIfFn:   defaultRetryIfFn,
		resetTimer:  make(chan time.Duration, 1),
	}
}

// New creates a new instance.
func New(httpClient HTTPClient, opts ...Option) (*HTTPRetrier, error) {
	c := defaultHTTPRetrier()

	for _, applyOpt := range opts {
		if err := applyOpt(c); err != nil {
			return nil, err
		}
	}

	c.httpClient = httpClient

	return c, nil
}

// Do attempts to run the request according to the retry rules.
func (c *HTTPRetrier) Do(r *http.Request) (*http.Response, error) {
	c.nextDelay = float64(c.delay)
	c.remainingAttempts = c.attempts
	c.ctx, c.cancel = context.WithCancel(context.Background())

	go c.retry(r)

	// wait for completion
	<-c.ctx.Done()

	return c.doResponse, c.doError
}

// defaultRetryIfFn is the default function to check the retry condition.
func defaultRetryIfFn(statusCode int, err error) bool {
	if err != nil {
		return true
	}

	switch statusCode {
	case http.StatusNotFound,
		http.StatusRequestTimeout,
		http.StatusConflict,
		http.StatusMisdirectedRequest,
		http.StatusTooEarly,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
		http.StatusInsufficientStorage:
		return true
	}

	return false
}

func (c *HTTPRetrier) setTimer(d time.Duration) {
	if !c.timer.Stop() {
		// make sure to drain timer channel before reset
		select {
		case <-c.timer.C:
		default:
		}
	}

	c.timer.Reset(d)
}

func (c *HTTPRetrier) retry(r *http.Request) {
	defer c.cancel()

	c.timer = time.NewTimer(1 * time.Nanosecond)

	for {
		select {
		case <-r.Context().Done():
			return
		case d := <-c.resetTimer:
			c.setTimer(d)
		case <-c.timer.C:
			if c.run(r) {
				return
			}
		}
	}
}

func (c *HTTPRetrier) run(r *http.Request) bool {
	c.doResponse, c.doError = c.httpClient.Do(r) // nolint:bodyclose

	c.remainingAttempts--
	if c.remainingAttempts == 0 {
		return true
	}

	var statusCode int
	if c.doResponse != nil {
		statusCode = c.doResponse.StatusCode
	}

	if !c.retryIfFn(statusCode, c.doError) {
		return true
	}

	if c.doError == nil {
		logging.Close(r.Context(), c.doResponse.Body, "error while closing response body")
	}

	c.resetTimer <- time.Duration(int64(c.nextDelay) + rand.Int63n(int64(c.jitter))) // nolint:gosec
	c.nextDelay *= c.delayFactor

	return false
}
