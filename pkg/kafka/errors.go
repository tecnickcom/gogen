package kafka

import "errors"

// Sentinel errors returned by the package. They can be matched with errors.Is
// so callers can distinguish configuration problems from runtime failures.
var (
	// ErrInvalidOptions is returned by NewConsumer and NewProducer when the
	// broker list, the topic, or a configured option value is missing or out
	// of range.
	ErrInvalidOptions = errors.New("kafka: missing or invalid client options")

	// ErrNilEncodeFunc is returned by NewProducer when the message encode function is nil.
	ErrNilEncodeFunc = errors.New("kafka: nil message encode function")

	// ErrNilDecodeFunc is returned by NewConsumer when the message decode function is nil.
	ErrNilDecodeFunc = errors.New("kafka: nil message decode function")

	// ErrConsumerClosed is returned by Receive, FetchMessage, and ReceiveData
	// after the consumer has been closed with Close.
	ErrConsumerClosed = errors.New("kafka: consumer closed")
)
