package kafka

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

type consumerMock struct {
	readMessage    func(ctx context.Context) (kafka.Message, error)
	fetchMessage   func(ctx context.Context) (kafka.Message, error)
	commitMessages func(ctx context.Context, msgs ...kafka.Message) error
	close          func() error
}

func (c consumerMock) ReadMessage(ctx context.Context) (kafka.Message, error) {
	return c.readMessage(ctx)
}

func (c consumerMock) FetchMessage(ctx context.Context) (kafka.Message, error) {
	return c.fetchMessage(ctx)
}

func (c consumerMock) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	return c.commitMessages(ctx, msgs...)
}

func (c consumerMock) Close() error {
	return c.close()
}

// newTestConsumer builds a Consumer with an injected mock reader, so no
// network client is constructed and no background goroutines are spawned.
func newTestConsumer(t *testing.T, mock consumerMock) *Consumer {
	t.Helper()

	consumer, err := NewConsumer(
		[]string{"url1:9092"},
		"topic1",
		"group1",
		WithKafkaReader(mock),
	)
	require.NoError(t, err)
	require.NotNil(t, consumer)

	return consumer
}

func Test_NewConsumer(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		brokers   []string
		topic     string
		groupID   string
		options   []Option
		wantErrIs error
	}{
		{
			name:    "success",
			brokers: []string{"url1:9092", "url2:9092"},
			topic:   "topic1",
			groupID: "one",
			options: []Option{
				WithSessionTimeout(time.Millisecond * 10),
				WithFirstOffset(),
			},
		},
		{
			name:    "success with injected reader",
			brokers: []string{"url1:9092"},
			topic:   "topic1",
			groupID: "one",
			options: []Option{
				WithKafkaReader(consumerMock{close: func() error { return nil }}),
			},
		},
		{
			name:      "empty broker list",
			brokers:   nil,
			topic:     "topic3",
			groupID:   "three",
			wantErrIs: ErrInvalidOptions,
		},
		{
			name:      "blank broker address",
			brokers:   []string{"url1:9092", " "},
			topic:     "topic1",
			groupID:   "one",
			wantErrIs: ErrInvalidOptions,
		},
		{
			name:    "missing decoding function",
			brokers: []string{"url1:9092", "url2:9092"},
			topic:   "topic1",
			groupID: "one",
			options: []Option{
				WithMessageDecodeFunc(nil),
			},
			wantErrIs: ErrNilDecodeFunc,
		},
		{
			name:    "negative session timeout",
			brokers: []string{"url1:9092"},
			topic:   "topic1",
			groupID: "one",
			options: []Option{
				WithSessionTimeout(-time.Second),
			},
			wantErrIs: ErrInvalidOptions,
		},
		{
			name:    "zero session timeout",
			brokers: []string{"url1:9092"},
			topic:   "topic1",
			groupID: "one",
			options: []Option{
				WithSessionTimeout(0),
			},
			wantErrIs: ErrInvalidOptions,
		},
		{
			name:    "session timeout too large",
			brokers: []string{"url1:9092"},
			topic:   "topic1",
			groupID: "one",
			options: []Option{
				WithSessionTimeout(25 * 24 * time.Hour),
			},
			wantErrIs: ErrInvalidOptions,
		},
		{
			name:      "empty topic",
			brokers:   []string{"url1:9092"},
			topic:     "",
			groupID:   "",
			wantErrIs: ErrInvalidOptions,
		},
		{
			name:    "empty topic rejected even with injected reader",
			brokers: []string{"url1:9092"},
			topic:   "",
			groupID: "one",
			options: []Option{
				WithKafkaReader(consumerMock{close: func() error { return nil }}),
			},
			wantErrIs: ErrInvalidOptions,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			consumer, err := NewConsumer(tt.brokers, tt.topic, tt.groupID, tt.options...)

			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
				require.Nil(t, consumer)
			} else {
				require.NoError(t, err)
				require.NotNil(t, consumer)
				require.NoError(t, consumer.Close())
			}
		})
	}
}

func Test_Consumer_Close(t *testing.T) {
	t.Parallel()

	consumer := newTestConsumer(t, consumerMock{
		close: func() error { return nil },
	})
	require.NoError(t, consumer.Close())

	consumer = newTestConsumer(t, consumerMock{
		close: func() error { return errors.New("close error") },
	})
	err := consumer.Close()
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot close the Kafka consumer")
}

func Test_Consumer_Receive(t *testing.T) {
	t.Parallel()

	errRead := errors.New("error Receive")

	testCases := []struct {
		name      string
		readErr   error
		wantMsg   []byte
		wantErrIs error
	}{
		{
			name:    "success",
			wantMsg: []byte{1},
		},
		{
			name:      "read error",
			readErr:   errRead,
			wantErrIs: errRead,
		},
		{
			name:      "closed consumer",
			readErr:   io.EOF,
			wantErrIs: ErrConsumerClosed,
		},
		{
			name:      "closed consumer with wrapped error",
			readErr:   fmt.Errorf("fetching message: %w", io.EOF),
			wantErrIs: ErrConsumerClosed,
		},
		{
			name:      "context canceled",
			readErr:   context.Canceled,
			wantErrIs: context.Canceled,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			consumer := newTestConsumer(t, consumerMock{
				readMessage: func(_ context.Context) (kafka.Message, error) {
					return kafka.Message{Value: tt.wantMsg}, tt.readErr
				},
			})

			msg, err := consumer.Receive(t.Context())

			if tt.wantErrIs == nil {
				require.NoError(t, err)
				require.Equal(t, tt.wantMsg, msg)

				return
			}

			require.ErrorIs(t, err, tt.wantErrIs)
			require.Nil(t, msg)
		})
	}
}

// Test_Consumer_Receive_dropsEOF asserts the H4 contract: after Close the raw
// io.EOF marker is replaced by ErrConsumerClosed and is no longer in the chain.
func Test_Consumer_Receive_dropsEOF(t *testing.T) {
	t.Parallel()

	consumer := newTestConsumer(t, consumerMock{
		readMessage: func(_ context.Context) (kafka.Message, error) {
			return kafka.Message{}, io.EOF
		},
		fetchMessage: func(_ context.Context) (kafka.Message, error) {
			return kafka.Message{}, io.EOF
		},
	})

	_, err := consumer.Receive(t.Context())
	require.ErrorIs(t, err, ErrConsumerClosed)
	require.NotErrorIs(t, err, io.EOF)

	_, err = consumer.FetchMessage(t.Context())
	require.ErrorIs(t, err, ErrConsumerClosed)
	require.NotErrorIs(t, err, io.EOF)
}

func Test_Consumer_FetchMessage(t *testing.T) {
	t.Parallel()

	errFetch := errors.New("error FetchMessage")

	testCases := []struct {
		name      string
		fetchErr  error
		wantErrIs error
	}{
		{
			name: "success",
		},
		{
			name:      "fetch error",
			fetchErr:  errFetch,
			wantErrIs: errFetch,
		},
		{
			name:      "closed consumer",
			fetchErr:  fmt.Errorf("fetching message: %w", io.EOF),
			wantErrIs: ErrConsumerClosed,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			consumer := newTestConsumer(t, consumerMock{
				fetchMessage: func(_ context.Context) (kafka.Message, error) {
					return kafka.Message{Value: []byte{1}}, tt.fetchErr
				},
			})

			msg, err := consumer.FetchMessage(t.Context())

			if tt.wantErrIs == nil {
				require.NoError(t, err)
				require.Equal(t, []byte{1}, msg.Value)

				return
			}

			require.ErrorIs(t, err, tt.wantErrIs)
			require.Empty(t, msg.Value)
		})
	}
}

func Test_Consumer_CommitMessages(t *testing.T) {
	t.Parallel()

	consumer := newTestConsumer(t, consumerMock{
		commitMessages: func(_ context.Context, _ ...kafka.Message) error { return nil },
	})
	err := consumer.CommitMessages(t.Context(), kafka.Message{Value: []byte{1}})
	require.NoError(t, err)

	consumer = newTestConsumer(t, consumerMock{
		commitMessages: func(_ context.Context, _ ...kafka.Message) error { return errors.New("error CommitMessages") },
	})
	err = consumer.CommitMessages(t.Context(), kafka.Message{Value: []byte{1}})
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot commit Kafka messages")
}

func Test_Consumer_FetchMessage_CommitMessages_flow(t *testing.T) {
	t.Parallel()

	var committed []kafka.Message

	consumer := newTestConsumer(t, consumerMock{
		fetchMessage: func(_ context.Context) (kafka.Message, error) {
			return kafka.Message{Topic: "topic1", Partition: 3, Offset: 42, Value: []byte("payload")}, nil
		},
		commitMessages: func(_ context.Context, msgs ...kafka.Message) error {
			committed = append(committed, msgs...)

			return nil
		},
	})

	ctx := t.Context()

	// fetch does not commit: the caller acknowledges after successful processing
	msg, err := consumer.FetchMessage(ctx)
	require.NoError(t, err)
	require.Equal(t, []byte("payload"), msg.Value)
	require.Empty(t, committed)

	err = consumer.CommitMessages(ctx, msg)
	require.NoError(t, err)
	require.Len(t, committed, 1)
	require.Equal(t, msg, committed[0])
}

func Test_Consumer_HealthCheck(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// every broker fails: the joined error names each address.
	consumer, err := NewConsumer(
		[]string{"bad1:9092", "bad2:9092"},
		"topic1",
		"group1",
		WithKafkaReader(consumerMock{close: func() error { return nil }}),
		WithBrokerCheckFunc(func(_ context.Context, address string) error {
			return fmt.Errorf("dial %s", address)
		}),
	)
	require.NoError(t, err)

	err = consumer.HealthCheck(ctx)
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot connect to Kafka")
	require.ErrorContains(t, err, "bad1:9092")
	require.ErrorContains(t, err, "bad2:9092")

	// the first broker fails and the second one succeeds.
	consumer, err = NewConsumer(
		[]string{"bad:9092", "good:9092"},
		"topic1",
		"group1",
		WithKafkaReader(consumerMock{close: func() error { return nil }}),
		WithBrokerCheckFunc(func(_ context.Context, address string) error {
			if address == "bad:9092" {
				return errors.New("dial error")
			}

			return nil
		}),
	)
	require.NoError(t, err)

	require.NoError(t, consumer.HealthCheck(ctx))
}

func Test_Consumer_Close_idempotent(t *testing.T) {
	t.Parallel()

	calls := 0

	consumer := newTestConsumer(t, consumerMock{
		close: func() error {
			calls++

			return nil
		},
	})

	require.NoError(t, consumer.Close())
	require.NoError(t, consumer.Close())
	require.Equal(t, 2, calls)
}

func TestReceiveData(t *testing.T) {
	t.Parallel()

	type TestData struct {
		Alpha string
		Beta  int
	}

	tests := []struct {
		name       string
		mock       consumerMock
		data       TestData
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "success",
			mock: consumerMock{
				readMessage: func(_ context.Context) (kafka.Message, error) {
					return kafka.Message{
						Value: []byte("Kf+BAwEBCFRlc3REYXRhAf+CAAECAQVBbHBoYQEMAAEEQmV0YQEEAAAAD/+CAQZhYmMxMjMB/gLtAA=="),
					}, nil
				},
			},
			data:    TestData{Alpha: "abc123", Beta: -375},
			wantErr: false,
		},
		{
			name: "empty",
			mock: consumerMock{
				readMessage: func(_ context.Context) (kafka.Message, error) { return kafka.Message{Value: []byte{}}, nil },
			},
			wantErr:    true,
			wantErrMsg: "cannot decode message data",
		},
		{
			name: "error",
			mock: consumerMock{
				readMessage: func(_ context.Context) (kafka.Message, error) { return kafka.Message{}, errors.New("error") },
			},
			wantErr:    true,
			wantErrMsg: "cannot read a message from Kafka",
		},
		{
			name: "invalid message",
			mock: consumerMock{
				readMessage: func(_ context.Context) (kafka.Message, error) {
					return kafka.Message{
						Value: []byte("你好世界"), //nolint:gosmopolitan
					}, nil
				},
			},
			wantErr:    true,
			wantErrMsg: "cannot decode message data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cli := newTestConsumer(t, tt.mock)

			var data TestData

			err := cli.ReceiveData(t.Context(), &data)
			if tt.wantErr {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErrMsg)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.data.Alpha, data.Alpha)
			require.Equal(t, tt.data.Beta, data.Beta)
		})
	}
}
