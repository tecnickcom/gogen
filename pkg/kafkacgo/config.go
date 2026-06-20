package kafkacgo

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// defaultFlushTimeoutMs is the default time Producer.Close waits for buffered
// messages to be delivered before closing the underlying client.
const defaultFlushTimeoutMs = 15_000

// config stores shared producer/consumer configuration and codec hooks.
type config struct {
	configMap         *kafka.ConfigMap
	messageEncodeFunc TEncodeFunc
	messageDecodeFunc TDecodeFunc
	flushTimeoutMs    int
}

// defaultConfig returns a config initialized with empty ConfigMap and default encode/decode functions.
func defaultConfig() *config {
	return &config{
		configMap:         &kafka.ConfigMap{},
		messageEncodeFunc: DefaultMessageEncodeFunc,
		messageDecodeFunc: DefaultMessageDecodeFunc,
		flushTimeoutMs:    defaultFlushTimeoutMs,
	}
}
