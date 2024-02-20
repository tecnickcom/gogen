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
	}

	for _, tt := range testCases {
		tt := tt

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

	ctx := context.TODO()

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

	ctx := context.TODO()

	consumer.client = &mockConsumerClient{}
	err = consumer.HealthCheck(ctx)
	require.Error(t, err)

	consumer.checkFn = func(_ context.Context, _ string) error {
		return nil
	}

	err = consumer.HealthCheck(ctx)
	require.NoError(t, err)
}
