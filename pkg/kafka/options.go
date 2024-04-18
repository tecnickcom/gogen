package kafka

import (
	"time"

	"github.com/segmentio/kafka-go"
)

// Option is a type alias for a function that configures Kafka client.
type Option func(*config)

// WithSessionTimeout sets the timeout used to detect client failures when using Kafka's group management facility.
// The client sends periodic heartbeats to indicate its liveness to the broker.
// If no heartbeats are received by the broker before the expiration of this session timeout,
// then the broker will remove this client from the group and initiate a rebalance.
func WithSessionTimeout(t time.Duration) Option {
	return func(c *config) {
		c.sessionTimeout = t
	}
}

// WithFirstOffset tells Kafka consumer to read from the beginning of uncommitted offset.
// By default it will read from the end of the queue.
func WithFirstOffset() Option {
	return func(c *config) {
		c.startOffset = kafka.FirstOffset
	}
}

// WithMessageEncodeFunc allow to replace DefaultMessageEncodeFunc.
// This function used by SendData() to encode the input data.
func WithMessageEncodeFunc(f TEncodeFunc) Option {
	return func(c *config) {
		c.messageEncodeFunc = f
	}
}

// WithMessageDecodeFunc allow to replace DefaultMessageDecodeFunc().
// This function used by ReceiveData() to decode a message encoded with messageEncodeFunc to the provided data object.
// The value underlying data must be a pointer to the correct type for the next data item received.
func WithMessageDecodeFunc(f TDecodeFunc) Option {
	return func(c *config) {
		c.messageDecodeFunc = f
	}
}
