package kafkacgo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Vonage/gosrvlib/pkg/encode"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// TDecodeFunc is the type of function used to replace the default message decoding function used by ReceiveData().
type TDecodeFunc func(ctx context.Context, msg []byte, data any) error

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

	if cfg.messageDecodeFunc == nil {
		return nil, errors.New("missing message decoding function")
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
	return c.client.Close() //nolint:wrapcheck
}

// Receive reads one message from the Kafka; is blocked if no messages in the queue.
func (c *Consumer) Receive() ([]byte, error) {
	msg, err := c.client.ReadMessage(-1)
	if err != nil {
		return nil, fmt.Errorf("failed to read kafka message: %w", err)
	}

	return msg.Value, nil
}

// DefaultMessageDecodeFunc is the default function to decode a message for ReceiveData().
// The value underlying data must be a pointer to the correct type for the next data item received.
func DefaultMessageDecodeFunc(_ context.Context, msg []byte, data any) error {
	return encode.ByteDecode(msg, data) //nolint:wrapcheck
}

// ReceiveData retrieves a message from the queue and extract its content in the data.
func (c *Consumer) ReceiveData(ctx context.Context, data any) error {
	message, err := c.Receive()
	if err != nil {
		return err
	}

	return c.cfg.messageDecodeFunc(ctx, message, data)
}
