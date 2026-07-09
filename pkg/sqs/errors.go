package sqs

import "errors"

// Exported sentinel errors returned by this package. Match them with errors.Is.
var (
	// ErrInvalidQueueURL is returned by New when queueURL is empty, unparseable,
	// or lacks a scheme or host.
	ErrInvalidQueueURL = errors.New("sqs: invalid queue URL")

	// ErrMissingMessageGroupID is returned by New when a FIFO queue URL is given
	// without a valid message group ID.
	ErrMissingMessageGroupID = errors.New("sqs: a valid message group ID is required for a FIFO queue")

	// ErrUnexpectedMessageGroupID is returned by New when a message group ID is
	// supplied for a standard (non-FIFO) queue, which does not use message groups.
	ErrUnexpectedMessageGroupID = errors.New("sqs: a message group ID must not be set for a standard queue")

	// ErrDedupIDNotAllowed is returned by SendWithDeduplicationID and
	// SendDataWithDeduplicationID when the target queue is not a FIFO queue.
	ErrDedupIDNotAllowed = errors.New("sqs: a message deduplication ID can only be used with FIFO queues")

	// ErrInvalidDedupID is returned by SendWithDeduplicationID and
	// SendDataWithDeduplicationID when the deduplication ID is empty or invalid.
	ErrInvalidDedupID = errors.New("sqs: invalid message deduplication ID")

	// ErrNilEncodeFunc is returned by New when the message encode function is nil.
	ErrNilEncodeFunc = errors.New("sqs: nil message encode function")

	// ErrNilDecodeFunc is returned by New when the message decode function is nil.
	ErrNilDecodeFunc = errors.New("sqs: nil message decode function")

	// ErrInvalidWaitTime is returned by New when waitTimeSeconds is outside the
	// valid 0..20 seconds range.
	ErrInvalidWaitTime = errors.New("sqs: waitTimeSeconds must be between 0 and 20 seconds")

	// ErrInvalidVisibilityTimeout is returned by New when visibilityTimeout is
	// outside the valid 0..43200 seconds range.
	ErrInvalidVisibilityTimeout = errors.New("sqs: visibilityTimeout must be between 0 and 43200 seconds")

	// ErrQueueNotResponding is returned by HealthCheck when the queue does not
	// return the expected attribute.
	ErrQueueNotResponding = errors.New("sqs: the queue is not responding")
)
