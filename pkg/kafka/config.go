package kafka

import (
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
