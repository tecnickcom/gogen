package kafka

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
)

type consumerClient interface {
	ReadMessage(ctx context.Context) (kafka.Message, error)
	Close() error
}

// Consumer represents a wrapper around kafka.Consumer.
type Consumer struct {
	cfg    *config
	client consumerClient
}

// NewConsumer creates a new instance of Consumer.
func NewConsumer(urls []string, topic, groupID string, opts ...Option) (*Consumer, error) {
	cfg := defaultConfig()

	for _, applyOpt := range opts {
		applyOpt(cfg)
	}

	params := kafka.ReaderConfig{
		Brokers:        urls,
		Topic:          topic,
		GroupID:        groupID,
		SessionTimeout: cfg.sessionTimeout,
		StartOffset:    cfg.startOffset,
	}

	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	return &Consumer{cfg: cfg, client: kafka.NewReader(params)}, nil
}

// Close cleans up Consumer's internal resources.
func (c *Consumer) Close() error {
	err := c.client.Close()
	if err != nil {
		return fmt.Errorf("failed to close the Kafka consumer: %w", err)
	}

	return nil
}

// Receive reads one message from the Kafka; blocks if there are no messages in the queue.
func (c *Consumer) Receive(ctx context.Context) ([]byte, error) {
	msg, err := c.client.ReadMessage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read a message from Kafka: %w", err)
	}

	return msg.Value, nil
}
