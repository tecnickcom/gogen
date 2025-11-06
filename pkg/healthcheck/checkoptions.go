package healthcheck

import (
	"net/http"
)

// checkConfig holds configuration options for healthchecks.
type checkConfig struct {
	configureRequest func(r *http.Request)
}

// CheckOption is a type alias for a function able to configure HTTP healthcheck options.
type CheckOption func(*checkConfig)

// WithConfigureRequest allows to configure the request before it is sent.
func WithConfigureRequest(fn func(r *http.Request)) CheckOption {
	return func(c *checkConfig) {
		c.configureRequest = fn
	}
}
