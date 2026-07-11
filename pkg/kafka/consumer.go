package kafka

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/segmentio/kafka-go"
	"github.com/tecnickcom/nurago/pkg/encode"
)

// TDecodeFunc is the type of function used to replace the default message decoding function used by ReceiveData().
type TDecodeFunc func(ctx context.Context, msg []byte, data any) error

// KReader defines the kafka-go reader methods used by [Consumer].
type KReader interface {
	ReadMessage(ctx context.Context) (kafka.Message, error)
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

// Consumer reads messages from a Kafka topic with pluggable decoding.
type Consumer struct {
	cfg     *config
	client  KReader
	checkFn checkBrokerFn
	brokers []string
}

// NewConsumer constructs a Kafka consumer for a topic and consumer group with optional tuning.
// Call HealthCheck() to verify broker/topic connectivity before beginning receives.
//
// groupID may be empty: without a consumer group the reader always starts
// from the earliest available offset (WithFirstOffset has no effect), offsets
// are never committed, and CommitMessages returns an error.
//
// Configuration problems are reported with errors matching ErrInvalidOptions
// or ErrNilDecodeFunc; match them with errors.Is.
func NewConsumer(brokers []string, topic, groupID string, opts ...Option) (*Consumer, error) {
	cfg := defaultConfig()

	for _, applyOpt := range opts {
		applyOpt(cfg)
	}

	err := cfg.validateConsumer(brokers, topic)
	if err != nil {
		return nil, err
	}

	client := cfg.reader

	if client == nil {
		// validateConsumer has already rejected everything kafka-go's
		// NewReader would panic on, so no separate ReaderConfig.Validate is
		// needed here.
		client = kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,
			SessionTimeout: cfg.sessionTimeout,
			StartOffset:    cfg.startOffset,
		})
	}

	checkFn := cfg.checkFn
	if checkFn == nil {
		checkFn = defaultCheckBroker(topic)
	}

	return &Consumer{
		cfg:     cfg,
		client:  client,
		checkFn: checkFn,
		brokers: slices.Clone(brokers),
	}, nil
}

// Close releases Consumer's resources and closes the broker connection.
// It is safe to call multiple times; receive methods called after Close
// return an error matching ErrConsumerClosed.
func (c *Consumer) Close() error {
	err := c.client.Close()
	if err != nil {
		return fmt.Errorf("cannot close the Kafka consumer: %w", err)
	}

	return nil
}

// Receive reads one message from the Kafka broker, blocking until a message arrives or ctx ends.
//
// Delivery semantics: at-most-once. When a consumer group is configured, the
// message offset is committed as soon as the message is read, before the
// caller processes it; if the process crashes (or decoding fails in
// ReceiveData) after Receive returns, the message is permanently skipped.
// For at-least-once semantics use FetchMessage and commit explicitly with
// CommitMessages after successful processing. Without a consumer group no
// offset is ever committed.
//
// After Close the returned error matches ErrConsumerClosed; when ctx is
// canceled or its deadline expires the returned error matches ctx.Err().
// Match both with errors.Is.
func (c *Consumer) Receive(ctx context.Context) ([]byte, error) {
	msg, err := c.client.ReadMessage(ctx)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("cannot read a message from Kafka: %w", ErrConsumerClosed)
		}

		return nil, fmt.Errorf("cannot read a message from Kafka: %w", err)
	}

	return msg.Value, nil
}

// FetchMessage reads one message from the Kafka broker without committing its
// offset, blocking until a message arrives or ctx ends.
//
// Delivery semantics: at-least-once. When a consumer group is configured, the
// caller must explicitly acknowledge the message by passing it to
// CommitMessages after successful processing; uncommitted messages are
// redelivered after a rebalance or restart. When no consumer group is
// configured, FetchMessage behaves like Receive.
//
// After Close the returned error matches ErrConsumerClosed; when ctx is
// canceled or its deadline expires the returned error matches ctx.Err().
// Match both with errors.Is.
func (c *Consumer) FetchMessage(ctx context.Context) (kafka.Message, error) {
	msg, err := c.client.FetchMessage(ctx)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return kafka.Message{}, fmt.Errorf("cannot fetch a message from Kafka: %w", ErrConsumerClosed)
		}

		return kafka.Message{}, fmt.Errorf("cannot fetch a message from Kafka: %w", err)
	}

	return msg, nil
}

// CommitMessages acknowledges (commits the offsets of) the messages returned
// by FetchMessage. It must be called only after the messages have been
// successfully processed, to obtain at-least-once delivery semantics.
// When no consumer group is configured, offset commits are unavailable and
// CommitMessages returns an error.
func (c *Consumer) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	err := c.client.CommitMessages(ctx, msgs...)
	if err != nil {
		return fmt.Errorf("cannot commit Kafka messages: %w", err)
	}

	return nil
}

// HealthCheck verifies broker reachability by probing each configured broker
// until one succeeds. When every broker fails, the individual probe errors
// are joined into the returned error.
//
// The default probe performs a partition lookup over plaintext TCP with the
// default kafka-go dialer, so it does not exercise TLS or SASL and always
// performs real network I/O even when a reader is injected via
// WithKafkaReader. Override it with WithBrokerCheckFunc to customize the
// dialer or to make HealthCheck deterministic in tests.
func (c *Consumer) HealthCheck(ctx context.Context) error {
	return healthCheck(ctx, c.brokers, c.checkFn)
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
//
// After Close the returned error matches ErrConsumerClosed — see Receive.
func (c *Consumer) ReceiveData(ctx context.Context, data any) error {
	message, err := c.Receive(ctx)
	if err != nil {
		return err
	}

	err = c.cfg.messageDecodeFunc(ctx, message, data)
	if err != nil {
		return fmt.Errorf("cannot decode message data: %w", err)
	}

	return nil
}
