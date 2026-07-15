package healthcheck

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// maxDrainBytes caps how much of an unread response body is drained before close.
// Draining lets keep-alive reuse the connection for small bodies; a larger body is
// only partially drained and its connection is dropped, which avoids an unbounded
// or hostile read.
const maxDrainBytes = 4 << 10

// HTTPClient is the minimal transport used by HTTP-based health checks.
type HTTPClient interface {
	// Do performs the HTTP request.
	Do(req *http.Request) (*http.Response, error)
}

// CheckHTTPStatus probes an HTTP endpoint and validates its response status.
//
// The check supports optional request mutation and returns an error for
// transport failures or mismatched status codes. By default the status must equal
// wantStatusCode; [WithAcceptStatus] replaces that with a predicate. A positive
// timeout bounds the request via a derived context; a non-positive timeout adds no
// deadline of its own, leaving only ctx to bound execution.
func CheckHTTPStatus(
	ctx context.Context,
	httpClient HTTPClient,
	method string,
	url string,
	wantStatusCode int,
	timeout time.Duration,
	opts ...CheckOption,
) (err error) {
	cfg := checkConfig{}

	for _, apply := range opts {
		apply(&cfg)
	}

	if timeout > 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	req, nerr := http.NewRequestWithContext(ctx, method, url, nil)
	if nerr != nil {
		return fmt.Errorf("build request: %w", nerr)
	}

	if cfg.configureRequest != nil {
		cfg.configureRequest(req)
	}

	resp, derr := httpClient.Do(req)
	if derr != nil {
		return fmt.Errorf("healthcheck request: %w", derr)
	}

	if resp == nil || resp.Body == nil {
		return errors.New("healthcheck request: nil response or body")
	}

	defer func() {
		// Drain a bounded amount of any remaining body so keep-alive can reuse the
		// connection for small bodies, without reading a large/hostile body in full.
		_, _ = io.CopyN(io.Discard, resp.Body, maxDrainBytes)
		err = errors.Join(err, resp.Body.Close())
	}()

	return checkStatus(resp.StatusCode, wantStatusCode, cfg.acceptStatus)
}

// checkStatus reports whether a response status is acceptable, using the optional
// predicate when set, otherwise an exact match against wantStatusCode.
func checkStatus(code, wantStatusCode int, accept func(code int) bool) error {
	if accept != nil {
		if accept(code) {
			return nil
		}

		return fmt.Errorf("unexpected healthcheck status code: got %d", code)
	}

	if code != wantStatusCode {
		return fmt.Errorf("unexpected healthcheck status code: got %d, want %d", code, wantStatusCode)
	}

	return nil
}
