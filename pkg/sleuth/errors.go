package sleuth

import "errors"

// Exported sentinel errors returned by this package. Match them with errors.Is.
var (
	// ErrInvalidAddress is returned by New when the Sleuth address is missing,
	// unparseable, or lacks a scheme or host.
	ErrInvalidAddress = errors.New("sleuth: invalid address")

	// ErrEmptyOrg is returned by New when the org slug is empty.
	ErrEmptyOrg = errors.New("sleuth: org is empty")

	// ErrEmptyAPIKey is returned by New when the API key is empty.
	ErrEmptyAPIKey = errors.New("sleuth: api key is empty")

	// ErrInvalidRetryConfig is returned by New when the retry options are invalid.
	ErrInvalidRetryConfig = errors.New("sleuth: invalid retry configuration")

	// ErrNilRequest is returned by the Send methods when the request is nil.
	ErrNilRequest = errors.New("sleuth: request must not be nil")
)
