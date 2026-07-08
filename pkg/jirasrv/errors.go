package jirasrv

import "errors"

// Exported sentinel errors returned by this package. Match them with errors.Is.
var (
	// ErrInvalidAddress is returned by New when the Jira address is missing,
	// unparseable, or lacks a scheme or host.
	ErrInvalidAddress = errors.New("jirasrv: invalid address")

	// ErrEmptyToken is returned by New when the bearer token is empty.
	ErrEmptyToken = errors.New("jirasrv: token is empty")

	// ErrInvalidRetryConfig is returned by New when the retry options are invalid.
	ErrInvalidRetryConfig = errors.New("jirasrv: invalid retry configuration")
)
