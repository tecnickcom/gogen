package kafkacgo

import (
	"fmt"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type consumerClient interface {
	ReadMessage(duration time.Duration) (*kafka.Message, error)
	Close() error
}

// Consumer represents a wrapper around kafka.Consumer.
type Consumer struct {
	cfg    *config
	client consumerClient
}

// NewConsumer creates a new instance of Consumer.
func NewConsumer(urls, topics []string, groupID string, opts ...Option) (*Consumer, error) {
	cfg := defaultConfig()

	for _, applyOpt := range opts {
		applyOpt(cfg)
	}

	_ = cfg.configMap.SetKey("bootstrap.servers", strings.Join(urls, ","))
	_ = cfg.configMap.SetKey("group.id", groupID)

	consumer, err := kafka.NewConsumer(cfg.configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create new kafka consumerClient: %w", err)
	}

	if err := consumer.SubscribeTopics(topics, nil); err != nil {
		return nil, fmt.Errorf("failed to subscribe kafka topic: %w", err)
	}

	return &Consumer{cfg: cfg, client: consumer}, nil
}

// Close cleans up Consumer's internal resources.
func (c *Consumer) Close() error {
	return c.client.Close() //nolint: wrapcheck
}

// Receive reads one message from the Kafka; is blocked if no messages in the queue.
func (c *Consumer) Receive() ([]byte, error) {
	msg, err := c.client.ReadMessage(-1)
	if err != nil {
		return nil, fmt.Errorf("failed to read kafka message: %w", err)
	}

	return msg.Value, nil
}
