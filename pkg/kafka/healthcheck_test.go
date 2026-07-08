package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test_defaultCheckBroker exercises the default probe closure without touching
// the network: an already-canceled context makes the partition lookup fail
// immediately.
func Test_defaultCheckBroker(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := defaultCheckBroker("topic")(ctx, "localhost:9092")
	require.Error(t, err)
}

func Test_healthCheck(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// first broker succeeds: later brokers are not probed.
	probed := make([]string, 0, 2)
	err := healthCheck(ctx, []string{"a:9092", "b:9092"}, func(_ context.Context, address string) error {
		probed = append(probed, address)

		return nil
	})
	require.NoError(t, err)
	require.Equal(t, []string{"a:9092"}, probed)

	// every broker fails: the joined error names each address.
	err = healthCheck(ctx, []string{"a:9092", "b:9092"}, func(_ context.Context, address string) error {
		return errors.New("dial " + address)
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot connect to Kafka")
	require.ErrorContains(t, err, "dial a:9092")
	require.ErrorContains(t, err, "dial b:9092")
}
