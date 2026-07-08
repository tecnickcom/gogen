package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

func Test_WithSessionTimeout(t *testing.T) {
	t.Parallel()

	v := time.Second * 17

	cfg := &config{}
	WithSessionTimeout(v)(cfg)
	require.Equal(t, v, cfg.sessionTimeout)
}

func Test_WithFirstOffset(t *testing.T) {
	t.Parallel()

	cfg := &config{}
	WithFirstOffset()(cfg)
	require.Equal(t, int64(-2), cfg.startOffset)
}

func Test_WithBalancer(t *testing.T) {
	t.Parallel()

	b := &kafka.RoundRobin{}

	cfg := &config{}
	WithBalancer(b)(cfg)
	require.Same(t, b, cfg.balancer)
}

func Test_WithRequiredAcks(t *testing.T) {
	t.Parallel()

	cfg := &config{}
	WithRequiredAcks(kafka.RequireOne)(cfg)
	require.Equal(t, kafka.RequireOne, cfg.requiredAcks)

	WithRequiredAcks(kafka.RequireAll)(cfg)
	require.Equal(t, kafka.RequireAll, cfg.requiredAcks)
}

func Test_WithBatchSize(t *testing.T) {
	t.Parallel()

	cfg := &config{}
	WithBatchSize(123)(cfg)
	require.Equal(t, 123, cfg.batchSize)
}

func Test_WithBatchTimeout(t *testing.T) {
	t.Parallel()

	v := time.Millisecond * 37

	cfg := &config{}
	WithBatchTimeout(v)(cfg)
	require.Equal(t, v, cfg.batchTimeout)
}

func Test_WithMessageEncodeFunc(t *testing.T) {
	t.Parallel()

	ret := []byte("test_data_001")
	f := func(_ context.Context, _ any) ([]byte, error) {
		return ret, nil
	}

	conf := &config{}
	WithMessageEncodeFunc(f)(conf)

	d, err := conf.messageEncodeFunc(t.Context(), "")
	require.NoError(t, err)
	require.Equal(t, ret, d)
}

func Test_WithMessageDecodeFunc(t *testing.T) {
	t.Parallel()

	f := func(_ context.Context, _ []byte, _ any) error {
		return nil
	}

	conf := &config{}
	WithMessageDecodeFunc(f)(conf)
	require.NoError(t, conf.messageDecodeFunc(t.Context(), nil, ""))
}

func Test_WithKafkaReader(t *testing.T) {
	t.Parallel()

	r := &consumerMock{}

	cfg := &config{}
	WithKafkaReader(r)(cfg)
	require.Same(t, r, cfg.reader)
}

func Test_WithKafkaWriter(t *testing.T) {
	t.Parallel()

	w := &produceMock{}

	cfg := &config{}
	WithKafkaWriter(w)(cfg)
	require.Same(t, w, cfg.writer)
}

func Test_WithBrokerCheckFunc(t *testing.T) {
	t.Parallel()

	called := false
	fn := func(_ context.Context, _ string) error {
		called = true

		return nil
	}

	cfg := &config{}
	WithBrokerCheckFunc(fn)(cfg)
	require.NotNil(t, cfg.checkFn)
	require.NoError(t, cfg.checkFn(t.Context(), "addr"))
	require.True(t, called)
}
