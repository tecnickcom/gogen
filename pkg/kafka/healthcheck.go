package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/segmentio/kafka-go"
)

const (
	// network is the network type used to connect to Kafka brokers.
	network = "tcp"
)

// checkBrokerFn is the type of function used to probe a single broker address.
type checkBrokerFn func(ctx context.Context, address string) error

// defaultCheckBroker returns the default broker probe: a partition lookup for
// topic performed with the default kafka-go dialer (the same dialer a
// non-customized reader uses).
func defaultCheckBroker(topic string) checkBrokerFn {
	return func(ctx context.Context, address string) error {
		_, err := kafka.DefaultDialer.LookupPartitions(ctx, network, address, topic)
		return err //nolint:wrapcheck
	}
}

// healthCheck probes each broker in order until one succeeds; when every
// broker fails, the individual probe errors are joined into the returned
// error.
func healthCheck(ctx context.Context, brokers []string, checkFn checkBrokerFn) error {
	var errs []error

	for _, address := range brokers {
		err := checkFn(ctx, address)
		if err == nil {
			return nil
		}

		errs = append(errs, err)
	}

	return fmt.Errorf("cannot connect to Kafka: %w", errors.Join(errs...))
}
