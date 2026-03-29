/*
Package httpretrier provides configurable retry execution for outbound HTTP
requests.

# Problem

Remote dependencies fail in transient ways (timeouts, connection resets,
temporary 5xx/429 responses). Without a consistent retry strategy, callers
either fail too aggressively or duplicate ad hoc retry loops with different
policies across services.

# Solution

This package wraps an [HTTPClient] with retry orchestration driven by a
pluggable [RetryIfFn].

[HTTPRetrier.Do] executes a request up to a bounded number of attempts,
applying delay growth and jitter between retries. The retry decision function
receives both `*http.Response` and `error`, enabling policy decisions based on
transport failures and/or HTTP status codes.

# Built-in Retry Policies

Predefined helpers are provided for common semantics:
  - [RetryIfForReadRequests] for idempotent reads (e.g. GET)
  - [RetryIfForWriteRequests] for state-changing writes (e.g. POST/PUT/PATCH)
  - [RetryIfFnByHTTPMethod] to select one of the above from method name

The default policy retries only when `err != nil`.

# Backoff Behavior

Delay progression is configurable via:
  - attempts cap ([WithAttempts])
  - initial delay ([WithDelay])
  - multiplicative delay factor ([WithDelayFactor])
  - random jitter ceiling ([WithJitter])

This produces bounded exponential-style backoff with randomization, helping
reduce synchronized retry storms.

# Request Body Replay

When a request has a body and retries are needed, the retrier relies on
`Request.GetBody` to recreate the body stream for subsequent attempts. If
GetBody is unavailable or fails, retries cannot continue and Do returns an
error.

# Benefits

httpretrier standardizes retry behavior for HTTP clients, improving resilience
while keeping retry policies explicit, testable, and reusable across services.
*/
package httpretrier

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
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

// RetryIfFn decides whether a request should be retried after a response/error.
type RetryIfFn func(r *http.Response, err error) bool

// HTTPClient is the minimal client contract used by [HTTPRetrier].
type HTTPClient interface {
	// Do performs the HTTP request.
	Do(req *http.Request) (*http.Response, error)
}

// HTTPRetrier applies configurable retry policies to HTTP requests.
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
	cancel            context.CancelFunc
	doResponse        *http.Response
	doError           error
}

// defaultHTTPRetrier creates a retrier instance with default values.
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

// New constructs HTTP retrier wrapping client with exponential backoff retry orchestration from provided options.
func New(httpClient HTTPClient, opts ...Option) (*HTTPRetrier, error) {
	c := defaultHTTPRetrier()

	for _, applyOpt := range opts {
		err := applyOpt(c)
		if err != nil {
			return nil, err
		}
	}

	c.httpClient = httpClient

	return c, nil
}

// Do executes request up to configured attempts with exponential delay and jitter between retries, applying retry decision function to each response.
func (c *HTTPRetrier) Do(r *http.Request) (*http.Response, error) {
	c.nextDelay = float64(c.delay)
	c.remainingAttempts = c.attempts
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel

	go c.retry(r)

	// wait for completion
	<-ctx.Done()

	return c.doResponse, c.doError
}

// defaultRetryIf is the default retry policy: returns true only when error is not nil (transport failures).
func defaultRetryIf(_ *http.Response, err error) bool {
	return err != nil
}

// RetryIfForWriteRequests is a retry policy for state-changing requests: retries on 429/502/503 or transport error.
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

// RetryIfForReadRequests is a retry policy for idempotent requests: retries on 404/408/409/423/425/429/500/502/503/504/507 or transport error.
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

// RetryIfFnByHTTPMethod selects retry policy by HTTP method: idempotent read policy for GET, write policy for all others.
func RetryIfFnByHTTPMethod(httpMethod string) RetryIfFn {
	if httpMethod == http.MethodGet {
		return RetryIfForReadRequests
	}

	return RetryIfForWriteRequests
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

// retry performs the retry logic.
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

// run performs a single attempt to execute the HTTP request.
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
		// we only close the body between attempts if there was no error
		cerr := c.doResponse.Body.Close()
		if cerr != nil {
			c.doError = fmt.Errorf("error while closing response body: %w", cerr)
			return true
		}
	}

	// set the original body for the next request
	r.Body = bodyRC

	c.resetTimer <- time.Duration(int64(c.nextDelay) + rand.Int63n(int64(c.jitter))) //nolint:gosec

	c.nextDelay *= c.delayFactor

	return false
}
