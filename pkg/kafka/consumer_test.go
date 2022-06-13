package kafka

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

func Test_NewConsumer(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		urls    []string
		topic   string
		groupID string
		options []Option
		wantErr bool
	}{
		{
			name:    "success",
			urls:    []string{"url1", "url2"},
			topic:   "topic1",
			groupID: "one",
			options: []Option{
				WithSessionTimeout(time.Millisecond * 10),
			},
			wantErr: false,
		},
	}

	for _, tt := range testCases {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			consumer, err := NewConsumer(tt.urls, tt.topic, tt.groupID, tt.options...)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, consumer)
			} else {
				require.NoError(t, err)
				require.NotNil(t, consumer)
				require.Nil(t, consumer.Close())
			}
		})
	}
}

type mockConsumerClient struct{}

func (m mockConsumerClient) ReadMessage(_ context.Context) (kafka.Message, error) {
	return kafka.Message{Value: []byte{1}}, nil
}

func (m mockConsumerClient) Close() error {
	return nil
}

type mockConsumerClientError struct{}

func (m mockConsumerClientError) ReadMessage(_ context.Context) (kafka.Message, error) {
	return kafka.Message{}, fmt.Errorf("error Receive")
}

func (m mockConsumerClientError) Close() error {
	return fmt.Errorf("error Close")
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

	consumer.client = mockConsumerClient{}
	msg, err := consumer.Receive(ctx)
	require.NoError(t, err)
	require.NotNil(t, msg)

	consumer.client = mockConsumerClientError{}
	msg, err = consumer.Receive(ctx)
	require.Error(t, err)
	require.Nil(t, msg)

	err = consumer.Close()
	require.Error(t, err)
}
