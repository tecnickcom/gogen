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

// Close releases producer resources and closes the underlying Kafka client.
func (p *Producer) Close() {
	p.client.Close()
}

// Send publishes a raw byte message to the specified Kafka topic.
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
