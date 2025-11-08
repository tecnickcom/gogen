package healthcheck

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// HTTPClient contains the function that performs the actual HTTP request.
type HTTPClient interface {
	// Do performs the HTTP request.
	Do(req *http.Request) (*http.Response, error)
}

// CheckHTTPStatus checks if the given HTTP request responds with the expected status code.
func CheckHTTPStatus(ctx context.Context, httpClient HTTPClient, method string, url string, wantStatusCode int, timeout time.Duration, opts ...CheckOption) (err error) {
	cfg := checkConfig{}

	for _, apply := range opts {
		apply(&cfg)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

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

	defer func() {
		err = errors.Join(err, resp.Body.Close())
	}()

	if resp.StatusCode != wantStatusCode {
		return fmt.Errorf("unexpected healthcheck status code: %d", resp.StatusCode)
	}

	return nil
}
