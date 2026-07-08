package devlake

import "errors"

// Exported sentinel errors returned by this package. Match them with errors.Is.
var (
	// ErrInvalidAddress is returned by New when the DevLake address is missing,
	// unparseable, or lacks a scheme or host.
	ErrInvalidAddress = errors.New("devlake: invalid address")

	// ErrEmptyAPIKey is returned by New when the API key is empty.
	ErrEmptyAPIKey = errors.New("devlake: api key is empty")

	// ErrInvalidRetryConfig is returned by New when the retry options are invalid.
	ErrInvalidRetryConfig = errors.New("devlake: invalid retry configuration")

	// ErrNilRequest is returned by the Send methods when the request is nil.
	ErrNilRequest = errors.New("devlake: request must not be nil")
)
