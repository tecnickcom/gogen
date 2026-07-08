package ipify

import "errors"

// Sentinel errors returned by the package. Match them with errors.Is so callers
// can distinguish configuration problems from response failures.
var (
	// ErrInvalidOptions is returned by New when the configured API URL is
	// missing, unparseable, or does not use an http/https scheme with a host.
	ErrInvalidOptions = errors.New("ipify: missing or invalid client options")

	// ErrInvalidResponse is returned by GetPublicIP when the endpoint returns a
	// nil response body or an empty (whitespace-only) payload.
	ErrInvalidResponse = errors.New("ipify: invalid response")
)
