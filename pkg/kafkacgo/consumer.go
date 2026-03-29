package kafkacgo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/tecnickcom/gogen/pkg/encode"
)

// TDecodeFunc is the type of function used to replace the default message decoding function used by ReceiveData().
type TDecodeFunc func(ctx context.Context, msg []byte, data any) error

// consumerClient captures the minimal consumer API used by [Consumer].
type consumerClient interface {
	ReadMessage(duration time.Duration) (*kafka.Message, error)
	Close() error
}

// Consumer reads and decodes messages from a configured Confluent Kafka consumer.
type Consumer struct {
	cfg    *config
	client consumerClient
}

// NewConsumer constructs a Kafka consumer subscribed to topics for the given group ID.
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

	err = consumer.SubscribeTopics(topics, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe kafka topic: %w", err)
	}

	return &Consumer{cfg: cfg, client: consumer}, nil
}

// Close releases consumer resources and closes the underlying Kafka client.
func (c *Consumer) Close() error {
	return c.client.Close() //nolint:wrapcheck
}

// Receive reads one Kafka message and blocks until one is available.
func (c *Consumer) Receive() ([]byte, error) {
	msg, err := c.client.ReadMessage(-1)
	if err != nil {
		return nil, fmt.Errorf("failed to read kafka message: %w", err)
	}

	return msg.Value, nil
}

// DefaultMessageDecodeFunc is the default ReceiveData deserializer using encode.ByteDecode.
// The data argument must be a pointer to the expected message type.
func DefaultMessageDecodeFunc(_ context.Context, msg []byte, data any) error {
	return encode.ByteDecode(msg, data) //nolint:wrapcheck
}

// ReceiveData receives a message and decodes it into data via the configured decode function.
func (c *Consumer) ReceiveData(ctx context.Context, data any) error {
	message, err := c.Receive()
	if err != nil {
		return err
	}

	return c.cfg.messageDecodeFunc(ctx, message, data)
}
