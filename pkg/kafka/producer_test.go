package kafka

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

type produceMock struct {
	writeMessages func(ctx context.Context, msgs ...kafka.Message) error
	close         func() error
}

func (p produceMock) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	return p.writeMessages(ctx, msgs...)
}

func (p produceMock) Close() error {
	return p.close()
}

// newTestProducer builds a Producer with an injected mock writer, so no
// network client is constructed.
func newTestProducer(t *testing.T, mock produceMock) *Producer {
	t.Helper()

	producer, err := NewProducer(
		[]string{"url1:9092"},
		"topic1",
		WithKafkaWriter(mock),
	)
	require.NoError(t, err)
	require.NotNil(t, producer)

	return producer
}

func Test_NewProducer(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		brokers         []string
		topic           string
		options         []Option
		expRequiredAcks kafka.RequiredAcks
		expBatchSize    int
		expBatchTimeout time.Duration
		injected        bool
		wantErrIs       error
	}{
		{
			name:    "success",
			brokers: []string{"url1:9092", "url2:9092"},
			topic:   "topic1",
			options: []Option{
				WithSessionTimeout(time.Millisecond * 17),
				WithBalancer(&kafka.RoundRobin{}),
			},
			expRequiredAcks: kafka.RequireAll,
			expBatchTimeout: defaultBatchTimeout,
		},
		{
			name:    "success with producer tuning options",
			brokers: []string{"url1:9092", "url2:9092"},
			topic:   "topic1",
			options: []Option{
				WithRequiredAcks(kafka.RequireOne),
				WithBatchSize(53),
				WithBatchTimeout(time.Millisecond * 21),
			},
			expRequiredAcks: kafka.RequireOne,
			expBatchSize:    53,
			expBatchTimeout: time.Millisecond * 21,
		},
		{
			name:    "success with injected writer",
			brokers: []string{"url1:9092"},
			topic:   "topic1",
			options: []Option{
				WithKafkaWriter(produceMock{close: func() error { return nil }}),
			},
			injected: true,
		},
		{
			name:    "missing encoding function",
			brokers: []string{"url1:9092", "url2:9092"},
			topic:   "topic1",
			options: []Option{
				WithMessageEncodeFunc(nil),
			},
			wantErrIs: ErrNilEncodeFunc,
		},
		{
			name:      "empty broker list",
			brokers:   nil,
			topic:     "topic1",
			wantErrIs: ErrInvalidOptions,
		},
		{
			name:      "blank broker address",
			brokers:   []string{"url1:9092", ""},
			topic:     "topic1",
			wantErrIs: ErrInvalidOptions,
		},
		{
			name:      "empty topic",
			brokers:   []string{"url1:9092"},
			topic:     "",
			wantErrIs: ErrInvalidOptions,
		},
		{
			name:    "negative batch size",
			brokers: []string{"url1:9092"},
			topic:   "topic1",
			options: []Option{
				WithBatchSize(-1),
			},
			wantErrIs: ErrInvalidOptions,
		},
		{
			name:    "zero batch timeout",
			brokers: []string{"url1:9092"},
			topic:   "topic1",
			options: []Option{
				WithBatchTimeout(0),
			},
			wantErrIs: ErrInvalidOptions,
		},
		{
			name:    "negative batch timeout",
			brokers: []string{"url1:9092"},
			topic:   "topic1",
			options: []Option{
				WithBatchTimeout(-time.Millisecond),
			},
			wantErrIs: ErrInvalidOptions,
		},
		{
			name:    "invalid required acks",
			brokers: []string{"url1:9092"},
			topic:   "topic1",
			options: []Option{
				WithRequiredAcks(kafka.RequiredAcks(5)),
			},
			wantErrIs: ErrInvalidOptions,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			producer, err := NewProducer(tt.brokers, tt.topic, tt.options...)

			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
				require.Nil(t, producer)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, producer)

			if !tt.injected {
				writer, ok := producer.client.(*kafka.Writer)
				require.True(t, ok)
				require.Equal(t, tt.expRequiredAcks, writer.RequiredAcks)
				require.Equal(t, tt.expBatchSize, writer.BatchSize)
				require.Equal(t, tt.expBatchTimeout, writer.BatchTimeout)
			}

			require.NoError(t, producer.Close())
		})
	}
}

func Test_Producer_Close(t *testing.T) {
	t.Parallel()

	producer := newTestProducer(t, produceMock{
		close: func() error { return nil },
	})
	require.NoError(t, producer.Close())

	producer = newTestProducer(t, produceMock{
		close: func() error { return errors.New("close error") },
	})
	err := producer.Close()
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot close the Kafka producer")
}

func TestSend(t *testing.T) {
	t.Parallel()

	producer := newTestProducer(t, produceMock{
		writeMessages: func(_ context.Context, _ ...kafka.Message) error { return nil },
	})
	err := producer.Send(t.Context(), []byte{1})
	require.NoError(t, err)

	producer = newTestProducer(t, produceMock{
		writeMessages: func(_ context.Context, _ ...kafka.Message) error { return errors.New("error WriteMessages") },
	})
	err = producer.Send(t.Context(), []byte{1})
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot send a message to Kafka")
}

func TestSendData(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	producer := newTestProducer(t, produceMock{
		writeMessages: func(_ context.Context, _ ...kafka.Message) error { return nil },
	})

	type TestData struct {
		Alpha string
		Beta  int
	}

	err := producer.SendData(ctx, TestData{Alpha: "abc345", Beta: -678})
	require.NoError(t, err)

	// the default encoder rejects nil data
	err = producer.SendData(ctx, nil)
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot encode message data for topic topic1")

	producer = newTestProducer(t, produceMock{
		writeMessages: func(_ context.Context, _ ...kafka.Message) error { return errors.New("error WriteMessages") },
	})
	err = producer.SendData(ctx, TestData{Alpha: "abc345", Beta: -678})
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot send a message to Kafka")
}

func Test_Producer_HealthCheck(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// every broker fails: the joined error names each address.
	producer, err := NewProducer(
		[]string{"bad1:9092", "bad2:9092"},
		"topic1",
		WithKafkaWriter(produceMock{close: func() error { return nil }}),
		WithBrokerCheckFunc(func(_ context.Context, address string) error {
			return fmt.Errorf("dial %s", address)
		}),
	)
	require.NoError(t, err)

	err = producer.HealthCheck(ctx)
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot connect to Kafka")
	require.ErrorContains(t, err, "bad1:9092")
	require.ErrorContains(t, err, "bad2:9092")

	// the first broker fails and the second one succeeds.
	producer, err = NewProducer(
		[]string{"bad:9092", "good:9092"},
		"topic1",
		WithKafkaWriter(produceMock{close: func() error { return nil }}),
		WithBrokerCheckFunc(func(_ context.Context, address string) error {
			if address == "bad:9092" {
				return errors.New("dial error")
			}

			return nil
		}),
	)
	require.NoError(t, err)

	require.NoError(t, producer.HealthCheck(ctx))
}

func Test_Producer_Close_idempotent(t *testing.T) {
	t.Parallel()

	calls := 0

	producer := newTestProducer(t, produceMock{
		close: func() error {
			calls++

			return nil
		},
	})

	require.NoError(t, producer.Close())
	require.NoError(t, producer.Close())
	require.Equal(t, 2, calls)
}
