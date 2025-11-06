package kafka

import (
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	defaultSessionTimeout = time.Second * 10
)

// config holds configuration options for the Kafka client.
type config struct {
	sessionTimeout    time.Duration
	startOffset       int64
	messageEncodeFunc TEncodeFunc
	messageDecodeFunc TDecodeFunc
}

// defaultConfig returns a config struct populated with default values.
func defaultConfig() *config {
	return &config{
		sessionTimeout:    defaultSessionTimeout,
		startOffset:       kafka.LastOffset,
		messageEncodeFunc: DefaultMessageEncodeFunc,
		messageDecodeFunc: DefaultMessageDecodeFunc,
	}
}
