package healthcheck

import (
	"net/http"
)

// checkConfig holds configuration options for healthchecks.
type checkConfig struct {
	configureRequest func(r *http.Request)
}

// CheckOption configures HTTP probe behavior used by [CheckHTTPStatus].
type CheckOption func(*checkConfig)

// WithConfigureRequest injects a request mutator before the probe is executed.
//
// This is useful for adding headers, auth tokens, or tracing fields to
// outbound healthcheck HTTP requests.
func WithConfigureRequest(fn func(r *http.Request)) CheckOption {
	return func(c *checkConfig) {
		c.configureRequest = fn
	}
}
