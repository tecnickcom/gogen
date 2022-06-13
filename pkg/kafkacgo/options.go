package kafkacgo

import (
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

const (
	// OffsetLatest automatically reset the offset to the latest offset.
	OffsetLatest Offset = "latest"

	// OffsetEarliest automatically reset the offset to the earliest offset.
	OffsetEarliest Offset = "earliest"

	// OffsetNone throw an error to the consumerClient if no previous offset is found for the consumerClient's group.
	OffsetNone Offset = "none"
)

// Offset points to where Kafka should start to read messages from.
type Offset string

// Option is a type alias for a function that configures Kafka client.
type Option func(*config)

// WithConfigParameter extends kafka.ConfigMap with additional parameters.
// Parameters are listed at:
// * consumer: https://docs.confluent.io/platform/current/installation/configuration/consumer-configs.html
// * producer: https://docs.confluent.io/platform/current/installation/configuration/producer-configs.html
func WithConfigParameter(key string, val kafka.ConfigValue) Option {
	return func(c *config) {
		_ = c.configMap.SetKey(key, val) // it never returns an error
	}
}

// WithSessionTimeout sets the timeout used to detect client failures when using Kafka's group management facility.
// The client sends periodic heartbeats to indicate its liveness to the broker.
// If no heartbeats are received by the broker before the expiration of this session timeout,
// then the broker will remove this client from the group and initiate a rebalance.
// Note that the value must be in the allowable range as configured in the broker configuration
// by group.min.session.timeout.ms and group.max.session.timeout.ms.
func WithSessionTimeout(t time.Duration) Option {
	return WithConfigParameter("session.timeout.ms", int(t.Milliseconds()))
}

// WithAutoOffsetResetPolicy sets what to do when there is no initial offset in Kafka
// or if the current offset does not exist any more on the server
// (e.g. because that data has been deleted).
func WithAutoOffsetResetPolicy(p Offset) Option {
	return WithConfigParameter("auto.offset.reset", string(p))
}

// WithProduceChannelSize sets the buffer size (in number of messages).
func WithProduceChannelSize(size int) Option {
	return WithConfigParameter("go.produce.channel.size", size)
}
