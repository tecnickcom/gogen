package kafka

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	defaultSessionTimeout = time.Second * 10

	// defaultBatchTimeout is the default maximum time the producer Writer waits
	// for a batch to fill before flushing it. Send()/SendData() publish one
	// message synchronously per call, so a small default keeps per-message
	// latency low (the kafka-go library default of 1s would make every Send
	// block up to ~1s). It can be overridden via WithBatchTimeout.
	defaultBatchTimeout = 10 * time.Millisecond

	// maxSessionTimeout is the exclusive upper bound for the consumer group
	// session timeout: kafka-go sends the value on the wire in milliseconds as
	// a signed 32-bit integer and panics inside NewReader when it is out of
	// range, so the bound is enforced at construction time instead.
	maxSessionTimeout = math.MaxInt32 * time.Millisecond
)

// config holds configuration options for the Kafka client.
type config struct {
	sessionTimeout    time.Duration
	startOffset       int64
	messageEncodeFunc TEncodeFunc
	messageDecodeFunc TDecodeFunc
	balancer          kafka.Balancer
	requiredAcks      kafka.RequiredAcks
	batchSize         int
	batchTimeout      time.Duration
	reader            KReader
	writer            KWriter
	checkFn           checkBrokerFn
}

// defaultConfig returns a config struct populated with default values.
func defaultConfig() *config {
	return &config{
		sessionTimeout:    defaultSessionTimeout,
		startOffset:       kafka.LastOffset,
		messageEncodeFunc: DefaultMessageEncodeFunc,
		messageDecodeFunc: DefaultMessageDecodeFunc,
		balancer:          &kafka.Hash{},
		requiredAcks:      kafka.RequireAll,
		batchTimeout:      defaultBatchTimeout,
	}
}

// validateBrokers checks that the broker address list is usable: it must be
// non-empty and contain no blank entries. Kafka accepts port-less host names
// (the default port 9092 applies), so entries are not required to be in
// host:port form; the connection-time validation is left to kafka-go.
func validateBrokers(brokers []string) error {
	if len(brokers) == 0 {
		return fmt.Errorf("empty broker list: %w", ErrInvalidOptions)
	}

	for _, addr := range brokers {
		if strings.TrimSpace(addr) == "" {
			return fmt.Errorf("blank broker address: %w", ErrInvalidOptions)
		}
	}

	return nil
}

// validateSessionTimeout checks the consumer group session timeout bounds:
// the timeout must be positive and below maxSessionTimeout. kafka-go only
// validates the value inside NewReader, where a violation is a panic, so it
// is checked here first.
func validateSessionTimeout(t time.Duration) error {
	if t <= 0 || t >= maxSessionTimeout {
		return fmt.Errorf("session timeout out of range (%s): %w", t, ErrInvalidOptions)
	}

	return nil
}

// validateTopic rejects an empty topic. Both constructors validate the topic
// with this helper so the error is identical on the consumer and producer.
func validateTopic(topic string) error {
	if topic == "" {
		return fmt.Errorf("empty topic: %w", ErrInvalidOptions)
	}

	return nil
}

// validateConsumer checks the consumer constructor arguments and the
// consumer-scoped configuration values. It validates everything kafka-go's
// NewReader would otherwise panic on (see maxSessionTimeout and the broker
// list) so the constructor can build the reader without a separate
// ReaderConfig.Validate call.
func (c *config) validateConsumer(brokers []string, topic string) error {
	if c.messageDecodeFunc == nil {
		return ErrNilDecodeFunc
	}

	err := validateBrokers(brokers)
	if err != nil {
		return err
	}

	err = validateTopic(topic)
	if err != nil {
		return err
	}

	return validateSessionTimeout(c.sessionTimeout)
}

// validateProducer checks the producer constructor arguments and the
// producer-scoped configuration values.
func (c *config) validateProducer(brokers []string, topic string) error {
	if c.messageEncodeFunc == nil {
		return ErrNilEncodeFunc
	}

	err := validateBrokers(brokers)
	if err != nil {
		return err
	}

	err = validateTopic(topic)
	if err != nil {
		return err
	}

	if c.batchSize < 0 {
		return fmt.Errorf("negative batch size (%d): %w", c.batchSize, ErrInvalidOptions)
	}

	// The kafka-go Writer silently falls back to its own 1s default when the
	// batch timeout is not positive, defeating the 10ms package default, so
	// non-positive values are rejected instead.
	if c.batchTimeout <= 0 {
		return fmt.Errorf("batch timeout out of range (%s): %w", c.batchTimeout, ErrInvalidOptions)
	}

	if c.requiredAcks != kafka.RequireNone && c.requiredAcks != kafka.RequireOne && c.requiredAcks != kafka.RequireAll {
		return fmt.Errorf("invalid required acks (%d): %w", c.requiredAcks, ErrInvalidOptions)
	}

	return nil
}
