package slack

import "errors"

// Exported sentinel errors returned by this package. Match them with errors.Is.
var (
	// ErrInvalidAddress is returned by New when the webhook address is missing,
	// unparseable, or lacks a scheme or host. The offending address is not
	// echoed in the wrapped error because the webhook URL is a secret.
	ErrInvalidAddress = errors.New("slack: invalid webhook address")

	// ErrInvalidRetryConfig is returned by New when the retry options are invalid.
	ErrInvalidRetryConfig = errors.New("slack: invalid retry configuration")
)
