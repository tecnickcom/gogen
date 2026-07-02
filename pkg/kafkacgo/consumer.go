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

// receivePollTimeout is the per-iteration timeout used by ReceiveCtx when polling
// ReadMessage. It bounds how often context cancellation is checked between reads.
const receivePollTimeout = 100 * time.Millisecond

// consumerClient captures the minimal consumer API used by [Consumer].
type consumerClient interface {
	SubscribeTopics(topics []string, rebalanceCb kafka.RebalanceCb) error
	ReadMessage(duration time.Duration) (*kafka.Message, error)
	CommitMessage(msg *kafka.Message) ([]kafka.TopicPartition, error)
	StoreMessage(msg *kafka.Message) ([]kafka.TopicPartition, error)
	Close() error
}

// Consumer reads and decodes messages from a configured Confluent Kafka consumer.
type Consumer struct {
	cfg    *config
	client consumerClient
}

// NewConsumer constructs a Kafka consumer subscribed to topics for the given group ID.
//
// Delivery semantics: with the librdkafka defaults (enable.auto.commit=true, 5s
// interval), offsets are committed on a timer regardless of processing outcome,
// so a crash between read and processing can permanently skip messages. For
// at-least-once processing either disable auto-commit via
// WithConfigParameter("enable.auto.commit", false) and call
// [Consumer.CommitMessage] after successful processing, or keep auto-commit
// enabled, set WithConfigParameter("enable.auto.offset.store", false), and call
// [Consumer.StoreMessage] after successful processing.
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

	return newConsumer(cfg, consumer, topics)
}

// newConsumer subscribes client to topics and wraps it in a [Consumer].
// On subscription failure the client is closed so librdkafka handles, sockets,
// and background threads do not leak on every failed construction.
func newConsumer(cfg *config, client consumerClient, topics []string) (*Consumer, error) {
	err := client.SubscribeTopics(topics, nil)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to subscribe kafka topic: %w", err), client.Close())
	}

	return &Consumer{cfg: cfg, client: client}, nil
}

// Close releases consumer resources and closes the underlying Kafka client.
func (c *Consumer) Close() error {
	return c.client.Close() //nolint:wrapcheck
}

// Receive reads one Kafka message and blocks until one is available.
//
// Receive does not honor any context and blocks indefinitely; prefer ReceiveCtx
// when cancellation or a deadline is required.
//
// See NewConsumer for the offset-commit delivery semantics; to acknowledge the
// message offset after processing, use ReceiveMessage with CommitMessage or
// StoreMessage instead.
func (c *Consumer) Receive() ([]byte, error) {
	msg, err := c.client.ReadMessage(-1)
	if err != nil {
		return nil, fmt.Errorf("failed to read kafka message: %w", err)
	}

	return msg.Value, nil
}

// ReceiveCtx reads one Kafka message, blocking until a message arrives or ctx is canceled.
//
// It polls ReadMessage with a short timeout in a loop: read timeouts are retried (after
// re-checking ctx), while any other read error is returned. When ctx is canceled before a
// message arrives, its error is returned wrapped.
//
// See NewConsumer for the offset-commit delivery semantics; to acknowledge the
// message offset after processing, use ReceiveMessage with CommitMessage or
// StoreMessage instead.
func (c *Consumer) ReceiveCtx(ctx context.Context) ([]byte, error) {
	msg, err := c.ReceiveMessage(ctx)
	if err != nil {
		return nil, err
	}

	return msg.Value, nil
}

// ReceiveMessage reads one Kafka message, blocking until a message arrives or ctx is
// canceled, and returns the full message so that its offset can be acknowledged after
// successful processing via CommitMessage or StoreMessage.
//
// It polls ReadMessage with a short timeout in a loop: read timeouts are retried (after
// re-checking ctx), while any other read error is returned. When ctx is canceled before a
// message arrives, its error is returned wrapped.
func (c *Consumer) ReceiveMessage(ctx context.Context) (*kafka.Message, error) {
	for {
		ctxErr := ctx.Err()
		if ctxErr != nil {
			return nil, fmt.Errorf("context canceled while reading kafka message: %w", ctxErr)
		}

		msg, err := c.client.ReadMessage(receivePollTimeout)
		if err != nil {
			var kerr kafka.Error
			if errors.As(err, &kerr) && kerr.IsTimeout() {
				continue
			}

			return nil, fmt.Errorf("failed to read kafka message: %w", err)
		}

		return msg, nil
	}
}

// CommitMessage synchronously commits the offset of msg (as returned by
// ReceiveMessage) to the consumer group. Call it after the message has been
// successfully processed to obtain at-least-once delivery semantics, with
// auto-commit disabled via WithConfigParameter("enable.auto.commit", false).
func (c *Consumer) CommitMessage(msg *kafka.Message) error {
	_, err := c.client.CommitMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to commit kafka message offset: %w", err)
	}

	return nil
}

// StoreMessage stores the offset of msg (as returned by ReceiveMessage) to be
// committed by the auto-commit timer. Call it after the message has been
// successfully processed, with automatic offset store disabled via
// WithConfigParameter("enable.auto.offset.store", false) and auto-commit left
// enabled, to obtain at-least-once delivery semantics without a synchronous
// commit per message.
func (c *Consumer) StoreMessage(msg *kafka.Message) error {
	_, err := c.client.StoreMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to store kafka message offset: %w", err)
	}

	return nil
}

// DefaultMessageDecodeFunc is the default ReceiveData deserializer using encode.ByteDecode.
// The data argument must be a pointer to the expected message type.
func DefaultMessageDecodeFunc(_ context.Context, msg []byte, data any) error {
	return encode.ByteDecode(msg, data) //nolint:wrapcheck
}

// ReceiveData receives a message (honoring ctx for cancellation) and decodes it into data
// via the configured decode function.
//
// See NewConsumer for the offset-commit delivery semantics; note that with the
// default auto-commit configuration a message whose decoding fails may still
// have its offset committed. For explicit acknowledgment use ReceiveMessage
// with CommitMessage or StoreMessage and decode the payload manually.
func (c *Consumer) ReceiveData(ctx context.Context, data any) error {
	message, err := c.ReceiveCtx(ctx)
	if err != nil {
		return err
	}

	return c.cfg.messageDecodeFunc(ctx, message, data)
}
