package kafka

import (
	"time"
)

const (
	defaultSessionTimeout = time.Second * 10
)

type config struct {
	sessionTimeout time.Duration
}

func defaultConfig() *config {
	return &config{
		sessionTimeout: defaultSessionTimeout,
	}
}
