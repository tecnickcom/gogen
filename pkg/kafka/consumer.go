package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/segmentio/kafka-go"
	"github.com/tecnickcom/gogen/pkg/encode"
	"go.uber.org/multierr"
)

const (
	// network is the network type used to connect to Kafka brokers.
	network = "tcp"
)

// TDecodeFunc is the type of function used to replace the default message decoding function used by ReceiveData().
type TDecodeFunc func(ctx context.Context, msg []byte, data any) error

type consumerClient interface {
	ReadMessage(ctx context.Context) (kafka.Message, error)
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

// Consumer reads messages from a Kafka topic with pluggable decoding.
type Consumer struct {
	cfg     *config
	client  consumerClient
	checkFn func(ctx context.Context, address string) error
	brokers []string
}

// NewConsumer constructs a Kafka consumer for a topic and consumer group with optional tuning.
// Call HealthCheck() to verify broker/topic connectivity before beginning receives.
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

	err := params.Validate()
	if err != nil {
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

// Close releases Consumer's resources and closes the broker connection.
func (c *Consumer) Close() error {
	err := c.client.Close()
	if err != nil {
		return fmt.Errorf("failed to close the Kafka consumer: %w", err)
	}

	return nil
}

// Receive reads one message from the Kafka broker, blocking until a message arrives or context cancels.
//
// Delivery semantics: at-most-once. When a consumer group is configured, the
// message offset is committed as soon as the message is read, before the
// caller processes it; if the process crashes (or decoding fails in
// ReceiveData) after Receive returns, the message is permanently skipped.
// For at-least-once semantics use FetchMessage and commit explicitly with
// CommitMessages after successful processing.
func (c *Consumer) Receive(ctx context.Context) ([]byte, error) {
	msg, err := c.client.ReadMessage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read a message from Kafka: %w", err)
	}

	return msg.Value, nil
}

// FetchMessage reads one message from the Kafka broker without committing its
// offset, blocking until a message arrives or context cancels.
//
// Delivery semantics: at-least-once. When a consumer group is configured, the
// caller must explicitly acknowledge the message by passing it to
// CommitMessages after successful processing; uncommitted messages are
// redelivered after a rebalance or restart. When no consumer group is
// configured, FetchMessage behaves like Receive.
func (c *Consumer) FetchMessage(ctx context.Context) (kafka.Message, error) {
	msg, err := c.client.FetchMessage(ctx)
	if err != nil {
		return kafka.Message{}, fmt.Errorf("failed to fetch a message from Kafka: %w", err)
	}

	return msg, nil
}

// CommitMessages acknowledges (commits the offsets of) the messages returned
// by FetchMessage. It must be called only after the messages have been
// successfully processed, to obtain at-least-once delivery semantics.
// It only applies when a consumer group is configured.
func (c *Consumer) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	err := c.client.CommitMessages(ctx, msgs...)
	if err != nil {
		return fmt.Errorf("failed to commit Kafka messages: %w", err)
	}

	return nil
}

// HealthCheck verifies broker reachability by attempting partition lookup on all configured brokers.
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

// DefaultMessageDecodeFunc is the default ReceiveData() deserializer, using encode.ByteDecode.
// The data argument must be a pointer matching the type of the decoded message.
func DefaultMessageDecodeFunc(_ context.Context, msg []byte, data any) error {
	return encode.ByteDecode(msg, data) //nolint:wrapcheck
}

// ReceiveData reads a message and decodes it into the provided data argument using the configured decoder.
//
// Delivery semantics: at-most-once — see Receive. In particular, when a
// consumer group is configured the offset is already committed when decoding
// happens, so a message failing to decode is permanently skipped. For
// at-least-once semantics use FetchMessage + CommitMessages and decode the
// payload manually.
func (c *Consumer) ReceiveData(ctx context.Context, data any) error {
	message, err := c.Receive(ctx)
	if err != nil {
		return err
	}

	return c.cfg.messageDecodeFunc(ctx, message, data)
}
