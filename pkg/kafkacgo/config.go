package kafkacgo

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type config struct {
	configMap         *kafka.ConfigMap
	messageEncodeFunc TEncodeFunc
	messageDecodeFunc TDecodeFunc
}

func defaultConfig() *config {
	return &config{
		configMap:         &kafka.ConfigMap{},
		messageEncodeFunc: DefaultMessageEncodeFunc,
		messageDecodeFunc: DefaultMessageDecodeFunc,
	}
}
