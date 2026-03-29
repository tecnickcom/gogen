package kafkacgo

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// config stores shared producer/consumer configuration and codec hooks.
type config struct {
	configMap         *kafka.ConfigMap
	messageEncodeFunc TEncodeFunc
	messageDecodeFunc TDecodeFunc
}

// defaultConfig returns a config initialized with empty ConfigMap and default encode/decode functions.
func defaultConfig() *config {
	return &config{
		configMap:         &kafka.ConfigMap{},
		messageEncodeFunc: DefaultMessageEncodeFunc,
		messageDecodeFunc: DefaultMessageDecodeFunc,
	}
}
