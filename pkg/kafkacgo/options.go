package kafkacgo

import (
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
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

// Option applies a configuration change shared by producer and consumer instances.
type Option func(*config)

// WithConfigParameter appends a raw librdkafka configuration key/value pair.
// Parameters are listed at:
// * consumer: https://docs.confluent.io/platform/current/installation/configuration/consumer-configs.html
// * producer: https://docs.confluent.io/platform/current/installation/configuration/producer-configs.html
func WithConfigParameter(key string, val kafka.ConfigValue) Option {
	return func(c *config) {
		_ = c.configMap.SetKey(key, val) // it never returns an error
	}
}

// WithSessionTimeout sets the consumer group session timeout used for heartbeat failure detection.
// The value must respect broker-side min/max session timeout configuration.
func WithSessionTimeout(t time.Duration) Option {
	return WithConfigParameter("session.timeout.ms", int(t.Milliseconds()))
}

// WithAutoOffsetResetPolicy sets behavior when no committed offset exists or stored offset is no longer available.
func WithAutoOffsetResetPolicy(p Offset) Option {
	return WithConfigParameter("auto.offset.reset", string(p))
}

// WithProduceChannelSize sets the internal producer channel buffer size in number of messages.
func WithProduceChannelSize(size int) Option {
	return WithConfigParameter("go.produce.channel.size", size)
}

// WithMessageEncodeFunc overrides DefaultMessageEncodeFunc used by SendData.
func WithMessageEncodeFunc(f TEncodeFunc) Option {
	return func(c *config) {
		c.messageEncodeFunc = f
	}
}

// WithMessageDecodeFunc overrides DefaultMessageDecodeFunc used by ReceiveData.
// The data argument passed to ReceiveData must be a pointer to the expected type.
func WithMessageDecodeFunc(f TDecodeFunc) Option {
	return func(c *config) {
		c.messageDecodeFunc = f
	}
}
