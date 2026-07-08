package kafka

import (
	"context"
	"fmt"
	"slices"

	"github.com/segmentio/kafka-go"
	"github.com/tecnickcom/gogen/pkg/encode"
)

// TEncodeFunc is the type of function used to replace the default message encoding function used by SendData().
type TEncodeFunc func(ctx context.Context, data any) ([]byte, error)

// KWriter defines the kafka-go writer methods used by [Producer].
type KWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

// Producer publishes messages to a Kafka topic with pluggable encoding.
type Producer struct {
	cfg     *config
	client  KWriter
	checkFn checkBrokerFn
	brokers []string
	topic   string
}

// NewProducer constructs a Kafka producer for a topic with optional tuning (encoding, balancing).
//
// The broker list and topic are validated at construction time: an empty or
// blank-entry broker list, an empty topic, a negative batch size, a
// non-positive batch timeout, and an unknown required-acks value are all
// rejected with errors matching ErrInvalidOptions (or ErrNilEncodeFunc for a
// nil encoder); match them with errors.Is.
//
// The partitioning strategy defaults to a kafka.Hash balancer. Because Send() publishes
// messages without a Key, the default Hash balancer concentrates all messages on a single
// partition; use WithBalancer to choose a different strategy (for example a round-robin
// balancer) to spread keyless messages across partitions.
//
// Broker acknowledgment defaults to kafka.RequireAll, so Send()/SendData() return an error
// when the write is not acknowledged by the full in-sync replica set; use WithRequiredAcks
// to trade durability for throughput.
//
// The batch flush timeout defaults to 10ms (instead of the kafka-go 1s default) to keep the
// latency of synchronous per-message Send() calls low; use WithBatchTimeout and
// WithBatchSize to tune batching behavior.
//
// The consumer-only options WithSessionTimeout and WithFirstOffset are accepted for API
// compatibility on the shared Option type but have no effect on a Producer.
func NewProducer(brokers []string, topic string, opts ...Option) (*Producer, error) {
	cfg := defaultConfig()

	for _, applyOpt := range opts {
		applyOpt(cfg)
	}

	err := cfg.validateProducer(brokers, topic)
	if err != nil {
		return nil, err
	}

	client := cfg.writer

	if client == nil {
		client = &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     cfg.balancer,
			RequiredAcks: cfg.requiredAcks,
			BatchSize:    cfg.batchSize,
			BatchTimeout: cfg.batchTimeout,
		}
	}

	checkFn := cfg.checkFn
	if checkFn == nil {
		checkFn = defaultCheckBroker(topic)
	}

	return &Producer{
		cfg:     cfg,
		client:  client,
		checkFn: checkFn,
		brokers: slices.Clone(brokers),
		topic:   topic,
	}, nil
}

// Close releases Producer's resources and closes the broker connection.
// It is safe to call multiple times.
func (p *Producer) Close() error {
	err := p.client.Close()
	if err != nil {
		return fmt.Errorf("cannot close the Kafka producer: %w", err)
	}

	return nil
}

// Send publishes a raw byte message to the Kafka topic.
func (p *Producer) Send(ctx context.Context, msg []byte) error {
	err := p.client.WriteMessages(
		ctx,
		kafka.Message{
			Value: msg,
		},
	)
	if err != nil {
		return fmt.Errorf("cannot send a message to Kafka: %w", err)
	}

	return nil
}

// HealthCheck verifies broker reachability by probing each configured broker
// until one succeeds. When every broker fails, the individual probe errors
// are joined into the returned error.
//
// The default probe performs a partition lookup over plaintext TCP with the
// default kafka-go dialer, so it does not exercise TLS or SASL and always
// performs real network I/O even when a writer is injected via
// WithKafkaWriter. Override it with WithBrokerCheckFunc to customize the
// dialer or to make HealthCheck deterministic in tests.
func (p *Producer) HealthCheck(ctx context.Context) error {
	return healthCheck(ctx, p.brokers, p.checkFn)
}

// DefaultMessageEncodeFunc is the default SendData() serializer, using encode.ByteEncode.
func DefaultMessageEncodeFunc(_ context.Context, data any) ([]byte, error) {
	return encode.ByteEncode(data) //nolint:wrapcheck
}

// SendData encodes the data argument and publishes the result to the Kafka topic using the configured encoder.
func (p *Producer) SendData(ctx context.Context, data any) error {
	message, err := p.cfg.messageEncodeFunc(ctx, data)
	if err != nil {
		return fmt.Errorf("cannot encode message data for topic %s: %w", p.topic, err)
	}

	return p.Send(ctx, message)
}
