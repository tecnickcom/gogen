package kafka

import (
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	defaultSessionTimeout = time.Second * 10
)

type config struct {
	sessionTimeout time.Duration
	startOffset    int64
}

func defaultConfig() *config {
	return &config{
		sessionTimeout: defaultSessionTimeout,
		startOffset:    kafka.LastOffset,
	}
}
