package kafka

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

func Test_NewConsumer(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		brokers []string
		topic   string
		groupID string
		options []Option
		wantErr bool
	}{
		{
			name:    "success",
			brokers: []string{"url1", "url2"},
			topic:   "topic1",
			groupID: "one",
			options: []Option{
				WithSessionTimeout(time.Millisecond * 10),
				WithFirstOffset(),
			},
			wantErr: false,
		},
		{
			name:    "invalid parameters",
			brokers: nil,
			topic:   "topic3",
			groupID: "three",
			wantErr: true,
		},
		{
			name:    "missing decoding function",
			brokers: []string{"url1", "url2"},
			topic:   "topic1",
			groupID: "one",
			options: []Option{
				WithMessageDecodeFunc(nil),
			},
			wantErr: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			consumer, err := NewConsumer(tt.brokers, tt.topic, tt.groupID, tt.options...)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, consumer)
			} else {
				require.NoError(t, err)
				require.NotNil(t, consumer)
				require.NoError(t, consumer.Close())
			}
		})
	}
}

type mockConsumerClient struct{}

func (m mockConsumerClient) ReadMessage(_ context.Context) (kafka.Message, error) {
	return kafka.Message{Value: []byte{1}}, nil
}

func (m mockConsumerClient) FetchMessage(_ context.Context) (kafka.Message, error) {
	return kafka.Message{Value: []byte{1}}, nil
}

func (m mockConsumerClient) CommitMessages(_ context.Context, _ ...kafka.Message) error {
	return nil
}

func (m mockConsumerClient) Config() kafka.ReaderConfig {
	return kafka.ReaderConfig{}
}

func (m mockConsumerClient) Close() error {
	return nil
}

type mockConsumerClientError struct{}

func (m mockConsumerClientError) ReadMessage(_ context.Context) (kafka.Message, error) {
	return kafka.Message{}, errors.New("error Receive")
}

func (m mockConsumerClientError) FetchMessage(_ context.Context) (kafka.Message, error) {
	return kafka.Message{}, errors.New("error FetchMessage")
}

func (m mockConsumerClientError) CommitMessages(_ context.Context, _ ...kafka.Message) error {
	return errors.New("error CommitMessages")
}

func (m mockConsumerClientError) Config() kafka.ReaderConfig {
	return kafka.ReaderConfig{}
}

func (m mockConsumerClientError) Close() error {
	return errors.New("error Close")
}

func Test_Consumer_Receive(t *testing.T) {
	t.Parallel()

	consumer, err := NewConsumer(
		[]string{"url1", "url2"},
		"topic1",
		"group1",
		WithSessionTimeout(time.Millisecond*10),
	)

	require.NoError(t, err)
	require.NotNil(t, consumer)

	ctx := t.Context()

	consumer.client = &mockConsumerClient{}
	msg, err := consumer.Receive(ctx)
	require.NoError(t, err)
	require.NotNil(t, msg)

	consumer.client = &mockConsumerClientError{}
	msg, err = consumer.Receive(ctx)
	require.Error(t, err)
	require.Nil(t, msg)

	err = consumer.Close()
	require.Error(t, err)
}

func Test_Consumer_FetchMessage(t *testing.T) {
	t.Parallel()

	consumer, err := NewConsumer(
		[]string{"url1", "url2"},
		"topic1",
		"group1",
	)

	require.NoError(t, err)
	require.NotNil(t, consumer)

	ctx := t.Context()

	consumer.client = &mockConsumerClient{}
	msg, err := consumer.FetchMessage(ctx)
	require.NoError(t, err)
	require.Equal(t, []byte{1}, msg.Value)

	consumer.client = &mockConsumerClientError{}
	_, err = consumer.FetchMessage(ctx)
	require.Error(t, err)
}

func Test_Consumer_CommitMessages(t *testing.T) {
	t.Parallel()

	consumer, err := NewConsumer(
		[]string{"url1", "url2"},
		"topic1",
		"group1",
	)

	require.NoError(t, err)
	require.NotNil(t, consumer)

	ctx := t.Context()

	consumer.client = &mockConsumerClient{}
	err = consumer.CommitMessages(ctx, kafka.Message{Value: []byte{1}})
	require.NoError(t, err)

	consumer.client = &mockConsumerClientError{}
	err = consumer.CommitMessages(ctx, kafka.Message{Value: []byte{1}})
	require.Error(t, err)
}

func Test_Consumer_FetchMessage_CommitMessages_flow(t *testing.T) {
	t.Parallel()

	consumer, err := NewConsumer(
		[]string{"url1", "url2"},
		"topic1",
		"group1",
	)

	require.NoError(t, err)
	require.NotNil(t, consumer)

	ctx := t.Context()

	var committed []kafka.Message

	consumer.client = consumerMock{
		fetchMessage: func(_ context.Context) (kafka.Message, error) {
			return kafka.Message{Topic: "topic1", Partition: 3, Offset: 42, Value: []byte("payload")}, nil
		},
		commitMessages: func(_ context.Context, msgs ...kafka.Message) error {
			committed = append(committed, msgs...)

			return nil
		},
		close: func() error { return nil },
	}

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

	consumer, err := NewConsumer(
		[]string{"url.invalid"},
		"topic2",
		"group2",
		WithSessionTimeout(time.Millisecond*13),
	)

	require.NoError(t, err)
	require.NotNil(t, consumer)

	ctx := t.Context()

	consumer.client = &mockConsumerClient{}
	err = consumer.HealthCheck(ctx)
	require.Error(t, err)

	consumer.checkFn = func(_ context.Context, _ string) error {
		return nil
	}

	err = consumer.HealthCheck(ctx)
	require.NoError(t, err)
}

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

func TestReceiveData(t *testing.T) {
	t.Parallel()

	type TestData struct {
		Alpha string
		Beta  int
	}

	tests := []struct {
		name    string
		mock    consumerClient
		data    TestData
		wantErr bool
	}{
		{
			name: "success",
			mock: consumerMock{
				readMessage: func(_ context.Context) (kafka.Message, error) {
					return kafka.Message{
						Value: []byte("Kf+BAwEBCFRlc3REYXRhAf+CAAECAQVBbHBoYQEMAAEEQmV0YQEEAAAAD/+CAQZhYmMxMjMB/gLtAA=="),
					}, nil
				},
				close: func() error { return nil },
			},
			data:    TestData{Alpha: "abc123", Beta: -375},
			wantErr: false,
		},
		{
			name: "empty",
			mock: consumerMock{
				readMessage: func(_ context.Context) (kafka.Message, error) { return kafka.Message{Value: []byte{}}, nil },
				close:       func() error { return nil },
			},
			wantErr: true,
		},
		{
			name: "error",
			mock: consumerMock{
				readMessage: func(_ context.Context) (kafka.Message, error) { return kafka.Message{}, errors.New("error") },
				close:       func() error { return nil },
			},
			wantErr: true,
		},
		{
			name: "invalid message",
			mock: consumerMock{
				readMessage: func(_ context.Context) (kafka.Message, error) {
					return kafka.Message{
						Value: []byte("你好世界"), //nolint:gosmopolitan
					}, nil
				},
				close: func() error { return nil },
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			cli, err := NewConsumer([]string{"url1", "url2"}, "topic", "groupID")
			require.NoError(t, err)
			require.NotNil(t, cli)

			cli.client = tt.mock

			var data TestData

			err = cli.ReceiveData(ctx, &data)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.data.Alpha, data.Alpha)
			require.Equal(t, tt.data.Beta, data.Beta)
		})
	}
}
