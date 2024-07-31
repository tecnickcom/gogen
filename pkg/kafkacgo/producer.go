package kafkacgo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Vonage/gosrvlib/pkg/encode"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// TEncodeFunc is the type of function used to replace the default message encoding function used by SendData().
type TEncodeFunc func(ctx context.Context, topic string, data any) ([]byte, error)

type producerClient interface {
	Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error
	Close()
}

// Producer represents a wrapper around kafka.Producer.
type Producer struct {
	cfg    *config
	client producerClient
}

// NewProducer creates a new instance of Producer.
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

// Close cleans up Producer's internal resources.
func (p *Producer) Close() {
	p.client.Close()
}

// Send sends a message to Kafka topic.
func (p *Producer) Send(topic string, msg []byte) error {
	err := p.client.Produce(
		&kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &topic,
				Partition: kafka.PartitionAny,
			},
			Value: msg,
		},
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to send a kafka message: %w", err)
	}

	return nil
}

// DefaultMessageEncodeFunc is the default function to encode the input data for SendData().
func DefaultMessageEncodeFunc(_ context.Context, _ string, data any) ([]byte, error) {
	return encode.ByteEncode(data) //nolint:wrapcheck
}

// SendData delivers the specified data as encoded message to the queue.
func (p *Producer) SendData(ctx context.Context, topic string, data any) error {
	message, err := p.cfg.messageEncodeFunc(ctx, topic, data)
	if err != nil {
		return err
	}

	return p.Send(topic, message)
}
