package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/Vonage/gosrvlib/pkg/typeutil"
	"github.com/segmentio/kafka-go"
)

// TEncodeFunc is the type of function used to replace the default message encoding function used by SendData().
type TEncodeFunc func(ctx context.Context, data any) ([]byte, error)

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

	if cfg.messageEncodeFunc == nil {
		return nil, errors.New("missing message encoding function")
	}

	producer := &kafka.Writer{
		Addr:     kafka.TCP(urls...),
		Topic:    topic,
		Balancer: &kafka.Hash{},
	}

	return &Producer{
		cfg:    cfg,
		client: producer,
	}, nil
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

// DefaultMessageEncodeFunc is the default function to encode the input data for SendData().
func DefaultMessageEncodeFunc(_ context.Context, data any) ([]byte, error) {
	return typeutil.ByteEncode(data) //nolint:wrapcheck
}

// SendData delivers the specified data as encoded message to the queue.
func (p *Producer) SendData(ctx context.Context, data any) error {
	message, err := p.cfg.messageEncodeFunc(ctx, data)
	if err != nil {
		return err
	}

	return p.Send(ctx, message)
}
