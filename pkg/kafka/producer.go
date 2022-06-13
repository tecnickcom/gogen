package kafka

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
)

type producerClient interface {
	WriteMessages(ctx context.Context, msg ...kafka.Message) error
	Close() error
}

// Producer represents a wrapper around kafka.Producer.
type Producer struct {
	cfg    *config
	client producerClient
}

// NewProducer creates a new instance of Producer.
func NewProducer(urls []string, topic string, opts ...Option) (*Producer, error) {
	cfg := defaultConfig()

	for _, applyOpt := range opts {
		applyOpt(cfg)
	}

	producer := &kafka.Writer{
		Addr:     kafka.TCP(urls...),
		Topic:    topic,
		Balancer: &kafka.Hash{},
	}

	return &Producer{cfg: cfg, client: producer}, nil
}

// Close cleans up Producer's internal resources.
func (p *Producer) Close() error {
	err := p.client.Close()
	if err != nil {
		return fmt.Errorf("failed to close the Kafka producer: %w", err)
	}

	return nil
}

// Send sends a message to Kafka topic.
func (p *Producer) Send(ctx context.Context, msg []byte) error {
	err := p.client.WriteMessages(
		ctx,
		kafka.Message{
			Value: msg,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to send a message to Kafka: %w", err)
	}

	return nil
}
