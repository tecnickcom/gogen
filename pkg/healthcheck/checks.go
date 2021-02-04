package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// HTTPClient contains the function that performs the actual HTTP request.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// CheckHTTPStatus checks if the given HTTP request responds with the expected status code.
func CheckHTTPStatus(ctx context.Context, httpClient HTTPClient, method string, url string, wantStatusCode int, timeout time.Duration, opts ...CheckOption) error {
	cfg := checkConfig{}

	for _, apply := range opts {
		apply(&cfg)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	if cfg.configureRequest != nil {
		cfg.configureRequest(req)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("healthcheck request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != wantStatusCode {
		return fmt.Errorf("unexpected healthcheck status code: %d", resp.StatusCode)
	}

	return nil
}
