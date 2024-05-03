package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/Vonage/gosrvlib/pkg/encode"
	"github.com/segmentio/kafka-go"
	"go.uber.org/multierr"
)

const (
	network = "tcp"
)

// TDecodeFunc is the type of function used to replace the default message decoding function used by ReceiveData().
type TDecodeFunc func(ctx context.Context, msg []byte, data any) error

type consumerClient interface {
	ReadMessage(ctx context.Context) (kafka.Message, error)
	Close() error
}

// Consumer represents a wrapper around kafka.Consumer.
type Consumer struct {
	cfg     *config
	client  consumerClient
	checkFn func(ctx context.Context, address string) error
	brokers []string
}

// NewConsumer creates a new instance of Consumer.
// Please call the HealthCheck() method to check if the connection is working.
func NewConsumer(brokers []string, topic, groupID string, opts ...Option) (*Consumer, error) {
	cfg := defaultConfig()

	for _, applyOpt := range opts {
		applyOpt(cfg)
	}

	if cfg.messageDecodeFunc == nil {
		return nil, errors.New("missing message decoding function")
	}

	params := kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		SessionTimeout: cfg.sessionTimeout,
		StartOffset:    cfg.startOffset,
	}

	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	client := kafka.NewReader(params)

	checkFn := func(ctx context.Context, address string) error {
		_, err := client.Config().Dialer.LookupPartitions(ctx, network, address, topic)
		return err //nolint:wrapcheck
	}

	return &Consumer{
		cfg:     cfg,
		client:  client,
		checkFn: checkFn,
		brokers: brokers,
	}, nil
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

// HealthCheck checks if the consumer is working.
func (c *Consumer) HealthCheck(ctx context.Context) error {
	var errors error

	for _, address := range c.brokers {
		err := c.checkFn(ctx, address)
		if err == nil {
			return nil
		}

		errors = multierr.Append(errors, err)
	}

	return fmt.Errorf("unable to connect to Kafka: %w", errors)
}

// DefaultMessageDecodeFunc is the default function to decode a message for ReceiveData().
// The value underlying data must be a pointer to the correct type for the next data item received.
func DefaultMessageDecodeFunc(_ context.Context, msg []byte, data any) error {
	return encode.ByteDecode(msg, data) //nolint:wrapcheck
}

// ReceiveData retrieves a message from the queue and extract its content in the data.
func (c *Consumer) ReceiveData(ctx context.Context, data any) error {
	message, err := c.Receive(ctx)
	if err != nil {
		return err
	}

	return c.cfg.messageDecodeFunc(ctx, message, data)
}
