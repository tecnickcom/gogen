package kafka

import (
	"time"
)

// Option is a type alias for a function that configures Kafka client.
type Option func(*config)

// WithSessionTimeout sets the timeout used to detect client failures when using Kafka's group management facility.
// The client sends periodic heartbeats to indicate its liveness to the broker.
// If no heartbeats are received by the broker before the expiration of this session timeout,
// then the broker will remove this client from the group and initiate a rebalance.
func WithSessionTimeout(t time.Duration) Option {
	return func(c *config) {
		c.sessionTimeout = t
	}
}
