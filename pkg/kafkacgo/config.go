package kafkacgo

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// config holds the configuration for the Kafka client.
type config struct {
	configMap         *kafka.ConfigMap
	messageEncodeFunc TEncodeFunc
	messageDecodeFunc TDecodeFunc
}

// defaultConfig returns a config instance with default settings.
func defaultConfig() *config {
	return &config{
		configMap:         &kafka.ConfigMap{},
		messageEncodeFunc: DefaultMessageEncodeFunc,
		messageDecodeFunc: DefaultMessageDecodeFunc,
	}
}
