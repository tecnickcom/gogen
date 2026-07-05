package healthcheck

import (
	"net/http"
)

// checkConfig holds configuration options for healthchecks.
type checkConfig struct {
	configureRequest func(r *http.Request)
	acceptStatus     func(code int) bool
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

// WithAcceptStatus replaces the exact status-code match in [CheckHTTPStatus] with
// a predicate, so a probe can accept a range of codes (for example any 2xx)
// instead of a single one. When set, the wantStatusCode argument is ignored.
func WithAcceptStatus(fn func(code int) bool) CheckOption {
	return func(c *checkConfig) {
		c.acceptStatus = fn
	}
}
