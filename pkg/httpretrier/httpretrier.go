// Package httpretrier allow to retry HTTP requests based on HTTP status code boolean conditions.
package httpretrier

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/Vonage/gosrvlib/pkg/logging"
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
type RetryIfFn func(r *http.Response, err error) bool

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
		retryIfFn:   defaultRetryIf,
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

// defaultRetryIf is the default function to check the retry condition.
func defaultRetryIf(_ *http.Response, err error) bool {
	return err != nil
}

// RetryIfForWriteRequests is a retry check function used for write requests
// (e.g. PUT/PATCH/POST requests that can modify the remote state).
func RetryIfForWriteRequests(r *http.Response, err error) bool {
	if err != nil {
		return true
	}

	switch r.StatusCode {
	case
		http.StatusTooManyRequests,    // 429
		http.StatusBadGateway,         // 502
		http.StatusServiceUnavailable: // 503
		return true
	}

	return false
}

// RetryIfForReadRequests is a retry check function used for read requests
// (e.g. GET requests that are guaranteed to not modify the remote state).
func RetryIfForReadRequests(r *http.Response, err error) bool {
	if err != nil {
		return true
	}

	switch r.StatusCode {
	case
		http.StatusNotFound,            // 404
		http.StatusRequestTimeout,      // 408
		http.StatusConflict,            // 409
		http.StatusLocked,              // 423
		http.StatusTooEarly,            // 425
		http.StatusTooManyRequests,     // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout,      // 504
		http.StatusInsufficientStorage: // 507
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
			c.doError = fmt.Errorf("request context has been canceled: %w", r.Context().Err())
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
	var (
		bodyRC io.ReadCloser
		err    error
	)

	if r.GetBody != nil {
		bodyRC, err = r.GetBody()
		if err != nil {
			c.doError = fmt.Errorf("error while reading request body: %w", err)
			return true
		}
	}

	c.doResponse, c.doError = c.httpClient.Do(r) //nolint:bodyclose

	c.remainingAttempts--
	if c.remainingAttempts == 0 || !c.retryIfFn(c.doResponse, c.doError) {
		return true
	}

	if c.doError == nil {
		// we only close the body between attempts
		logging.Close(r.Context(), c.doResponse.Body, "error while closing response body")
	}

	// set the original body for the next request
	r.Body = bodyRC

	c.resetTimer <- time.Duration(int64(c.nextDelay) + rand.Int63n(int64(c.jitter))) //nolint:gosec
	c.nextDelay *= c.delayFactor

	return false
}
