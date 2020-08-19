package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// CheckHTTPStatus checks if the given HTTP request responds with the expected status code
func CheckHTTPStatus(ctx context.Context, method string, url string, wantStatusCode int, timeout time.Duration) error {
	httpClient := http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %v", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("healthcheck request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != wantStatusCode {
		return fmt.Errorf("unexpected healthcheck status code: %d", resp.StatusCode)
	}
	return nil
}
