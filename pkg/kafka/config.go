package kafka

import (
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type config struct {
	configMap *kafka.ConfigMap
}

func defaultConfig() *config {
	return &config{
		configMap: &kafka.ConfigMap{},
	}
}
