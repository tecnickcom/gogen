package kafka

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
)

// Option configures Kafka producer/consumer behavior.
type Option func(*config)

// WithSessionTimeout customizes the heartbeat timeout for broker group-management failure detection.
// Defaults to 10 seconds if not set.
//
// The value must be positive and below math.MaxInt32 milliseconds (~24.8
// days); NewConsumer rejects out-of-range values with an error matching
// ErrInvalidOptions. It only takes effect when a consumer group is
// configured; without a group ID the value is validated but unused.
//
// This option is consumer-only and has no effect on a Producer; passing it to
// NewProducer is silently ignored.
func WithSessionTimeout(t time.Duration) Option {
	return func(c *config) {
		c.sessionTimeout = t
	}
}

// WithFirstOffset makes a consumer group without a committed offset start
// consumption from the earliest available offset instead of the default
// latest. It only applies when a consumer group is configured: without a
// group ID this option has no effect and reading always starts from the
// earliest available offset.
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
// Any other value is rejected by NewProducer with an error matching
// ErrInvalidOptions.
//
// This option is producer-only and has no effect on a Consumer; passing it to
// NewConsumer is silently ignored.
func WithRequiredAcks(acks kafka.RequiredAcks) Option {
	return func(c *config) {
		c.requiredAcks = acks
	}
}

// WithBatchSize sets the maximum number of messages the Producer buffers into a
// single batch before sending it to the brokers. If not set (or set to 0), the
// kafka-go library default (100) is used. Negative values are rejected by
// NewProducer with an error matching ErrInvalidOptions.
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
// The value must be positive: the kafka-go Writer silently replaces a
// non-positive timeout with its own 1s default, so NewProducer rejects such
// values with an error matching ErrInvalidOptions.
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

// WithKafkaReader injects an existing kafka-go reader (or a compatible
// implementation), primarily for testing.
//
// When a reader is injected, NewConsumer does not build a kafka.ReaderConfig:
// the groupID argument is not used for reading, and the session timeout and
// start offset settings have no effect (their values are still validated).
// The brokers and topic arguments remain required (and validated) because
// HealthCheck probes them; note that injecting a reader does NOT mock
// HealthCheck, which still performs real network I/O unless WithBrokerCheckFunc
// is also supplied.
//
// This option is consumer-only and has no effect on a Producer; passing it to
// NewProducer is silently ignored.
func WithKafkaReader(r KReader) Option {
	return func(c *config) {
		c.reader = r
	}
}

// WithKafkaWriter injects an existing kafka-go writer (or a compatible
// implementation), primarily for testing.
//
// When a writer is injected, NewProducer does not construct a kafka.Writer:
// the balancer, required acks, and batch settings have no effect (their
// values are still validated). The brokers and topic arguments remain
// required (and validated) because HealthCheck probes them and the topic
// appears in encode error messages; note that injecting a writer does NOT
// mock HealthCheck, which still performs real network I/O unless
// WithBrokerCheckFunc is also supplied.
//
// This option is producer-only and has no effect on a Consumer; passing it to
// NewConsumer is silently ignored.
func WithKafkaWriter(w KWriter) Option {
	return func(c *config) {
		c.writer = w
	}
}

// WithBrokerCheckFunc overrides the per-broker reachability probe used by
// Consumer.HealthCheck and Producer.HealthCheck. fn is called with each
// configured broker address until one returns nil.
//
// The default probe performs a partition lookup over plaintext TCP with the
// default kafka-go dialer. Override it to use a custom dialer (for example
// with TLS or SASL), or to make HealthCheck deterministic and network-free in
// tests when a client is injected via WithKafkaReader / WithKafkaWriter.
func WithBrokerCheckFunc(fn func(ctx context.Context, address string) error) Option {
	return func(c *config) {
		c.checkFn = fn
	}
}
