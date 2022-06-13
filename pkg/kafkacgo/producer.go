package kafkacgo

import (
	"fmt"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type producerClient interface {
	Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error
	Close()
}

// Producer represents a wrapper around kafka.Producer.
type Producer struct {
	cfg    *config
	client producerClient
}

// NewProducer creates a new instance of Producer.
func NewProducer(urls []string, opts ...Option) (*Producer, error) {
	cfg := defaultConfig()

	for _, applyOpt := range opts {
		applyOpt(cfg)
	}

	_ = cfg.configMap.SetKey("bootstrap.servers", strings.Join(urls, ","))

	producer, err := kafka.NewProducer(cfg.configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create new kafka producer: %w", err)
	}

	return &Producer{cfg: cfg, client: producer}, nil
}

// Close cleans up Producer's internal resources.
func (p *Producer) Close() {
	p.client.Close()
}

// Send sends a message to Kafka topic.
func (p *Producer) Send(topic string, msg []byte) error {
	err := p.client.Produce(
		&kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &topic,
				Partition: kafka.PartitionAny,
			},
			Value: msg,
		},
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to send a kafka message: %w", err)
	}

	return nil
}
