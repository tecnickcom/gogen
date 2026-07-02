package kafkacgo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/tecnickcom/gogen/pkg/encode"
)

// TEncodeFunc is the type of function used to replace the default message encoding function used by SendData().
type TEncodeFunc func(ctx context.Context, topic string, data any) ([]byte, error)

// producerClient captures the minimal producer API used by [Producer].
type producerClient interface {
	Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error
	Flush(timeoutMs int) int
	Close()
}

// Producer publishes messages through a configured Confluent Kafka producer.
type Producer struct {
	cfg    *config
	client producerClient
}

// NewProducer constructs a Kafka producer with optional configuration and custom message encoder.
func NewProducer(urls []string, opts ...Option) (*Producer, error) {
	cfg := defaultConfig()

	for _, applyOpt := range opts {
		applyOpt(cfg)
	}

	if cfg.messageEncodeFunc == nil {
		return nil, errors.New("missing message encoding function")
	}

	_ = cfg.configMap.SetKey("bootstrap.servers", strings.Join(urls, ","))

	producer, err := kafka.NewProducer(cfg.configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create new kafka producer: %w", err)
	}

	return &Producer{cfg: cfg, client: producer}, nil
}

// Close flushes any buffered messages (up to the configured flush timeout) and
// then releases producer resources and closes the underlying Kafka client.
//
// It returns an error when events (undelivered messages, outstanding requests,
// or unread delivery reports) are still queued after the flush timeout
// expires: those events are dropped when the client is closed.
func (p *Producer) Close() error {
	remaining := p.client.Flush(p.cfg.flushTimeoutMs)
	p.client.Close()

	if remaining > 0 {
		return fmt.Errorf("kafka producer closed with %d unflushed events still in queue", remaining)
	}

	return nil
}

// Send publishes a raw byte message to the specified Kafka topic and blocks
// until the broker confirms delivery, returning an error if delivery fails.
func (p *Producer) Send(topic string, msg []byte) error {
	// Per-message delivery channel: librdkafka reports the delivery outcome here.
	// Without it, Produce only enqueues and delivery failures are silently lost.
	deliveryChan := make(chan kafka.Event, 1)
	defer close(deliveryChan)

	err := p.client.Produce(
		&kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &topic,
				Partition: kafka.PartitionAny,
			},
			Value: msg,
		},
		deliveryChan,
	)
	if err != nil {
		return fmt.Errorf("failed to send a kafka message: %w", err)
	}

	ev := <-deliveryChan

	m, ok := ev.(*kafka.Message)
	if !ok {
		return fmt.Errorf("unexpected kafka delivery event: %T", ev)
	}

	if m.TopicPartition.Error != nil {
		return fmt.Errorf("failed to deliver a kafka message: %w", m.TopicPartition.Error)
	}

	return nil
}

// DefaultMessageEncodeFunc is the default SendData serializer using encode.ByteEncode.
func DefaultMessageEncodeFunc(_ context.Context, _ string, data any) ([]byte, error) {
	return encode.ByteEncode(data) //nolint:wrapcheck
}

// SendData encodes data and publishes the result to topic.
func (p *Producer) SendData(ctx context.Context, topic string, data any) error {
	message, err := p.cfg.messageEncodeFunc(ctx, topic, data)
	if err != nil {
		return err
	}

	return p.Send(topic, message)
}
