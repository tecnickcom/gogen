package kafka

import (
	"time"

	"github.com/segmentio/kafka-go"
)

// Option configures Kafka producer/consumer behavior.
type Option func(*config)

// WithSessionTimeout customizes the heartbeat timeout for broker group-management failure detection.
// Defaults to 10 seconds if not set.
//
// This option is consumer-only and has no effect on a Producer; passing it to
// NewProducer is silently ignored.
func WithSessionTimeout(t time.Duration) Option {
	return func(c *config) {
		c.sessionTimeout = t
	}
}

// WithFirstOffset starts consumption from the earliest available offset instead of the default latest.
//
// This option is consumer-only and has no effect on a Producer; passing it to
// NewProducer is silently ignored.
func WithFirstOffset() Option {
	return func(c *config) {
		c.startOffset = kafka.FirstOffset
	}
}

// WithBalancer sets the partitioning strategy (kafka.Balancer) used by the Producer
// to assign messages to partitions. If not set, the default is a kafka.Hash balancer.
//
// NOTE: the default kafka.Hash balancer partitions by message Key. Because Send()
// publishes messages without a Key, the default concentrates all messages on a single
// partition. Use WithBalancer (for example with &kafka.RoundRobin{} or &kafka.LeastBytes{})
// to distribute keyless messages across partitions.
//
// This option is producer-only and has no effect on a Consumer; passing it to
// NewConsumer is silently ignored.
func WithBalancer(b kafka.Balancer) Option {
	return func(c *config) {
		c.balancer = b
	}
}

// WithRequiredAcks sets the number of broker acknowledgments required before
// a Producer write is considered successful:
//
//   - kafka.RequireNone (0): fire-and-forget, broker-side failures are silently lost;
//   - kafka.RequireOne (1): wait for the partition leader to acknowledge the write;
//   - kafka.RequireAll (-1): wait for the full in-sync replica set (default).
//
// This option is producer-only and has no effect on a Consumer; passing it to
// NewConsumer is silently ignored.
func WithRequiredAcks(acks kafka.RequiredAcks) Option {
	return func(c *config) {
		c.requiredAcks = acks
	}
}

// WithBatchSize sets the maximum number of messages the Producer buffers into a
// single batch before sending it to the brokers. If not set, the kafka-go
// library default (100) is used.
//
// This option is producer-only and has no effect on a Consumer; passing it to
// NewConsumer is silently ignored.
func WithBatchSize(size int) Option {
	return func(c *config) {
		c.batchSize = size
	}
}

// WithBatchTimeout sets the maximum time an incomplete message batch is
// buffered before being flushed to the brokers. Because Send() and SendData()
// publish one message synchronously per call, each call can block up to this
// duration; it defaults to 10ms to keep per-message latency low.
//
// This option is producer-only and has no effect on a Consumer; passing it to
// NewConsumer is silently ignored.
func WithBatchTimeout(t time.Duration) Option {
	return func(c *config) {
		c.batchTimeout = t
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
