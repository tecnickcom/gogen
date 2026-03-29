package kafka

import (
	"time"

	"github.com/segmentio/kafka-go"
)

// Option configures Kafka producer/consumer behavior.
type Option func(*config)

// WithSessionTimeout customizes the heartbeat timeout for broker group-management failure detection.
// Defaults to 10 seconds if not set.
func WithSessionTimeout(t time.Duration) Option {
	return func(c *config) {
		c.sessionTimeout = t
	}
}

// WithFirstOffset starts consumption from the earliest available offset instead of the default latest.
func WithFirstOffset() Option {
	return func(c *config) {
		c.startOffset = kafka.FirstOffset
	}
}

// WithMessageEncodeFunc overrides DefaultMessageEncodeFunc for SendData() serialization.
func WithMessageEncodeFunc(f TEncodeFunc) Option {
	return func(c *config) {
		c.messageEncodeFunc = f
	}
}

// WithMessageDecodeFunc overrides DefaultMessageDecodeFunc for ReceiveData() deserialization.
// The data argument to ReceiveData() must be a pointer to the correct type.
func WithMessageDecodeFunc(f TDecodeFunc) Option {
	return func(c *config) {
		c.messageDecodeFunc = f
	}
}
