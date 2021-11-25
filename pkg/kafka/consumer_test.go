package kafka

import (
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/stretchr/testify/require"
)

func TestConsumer(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                          string
		urls                          []string
		topics                        []string
		groupID                       string
		options                       []Option
		expectedTimeout               time.Duration
		expectedAutoOffsetResetPolicy Offset
		expectErr                     bool
	}{
		{
			name:    "success",
			urls:    []string{"url1", "url2"},
			topics:  []string{"topic1", "topic2"},
			groupID: "one",
			options: []Option{
				WithSessionTimeout(time.Second * 10),
				WithAutoOffsetResetPolicy(OffsetLatest),
			},
			expectedTimeout:               time.Second * 10,
			expectedAutoOffsetResetPolicy: OffsetLatest,
			expectErr:                     false,
		},
		{
			name:    "bad offset",
			urls:    []string{"url1", "url2"},
			topics:  []string{"topic1", "topic2"},
			groupID: "one",
			options: []Option{
				WithAutoOffsetResetPolicy("bad offset"),
			},
			expectErr: true,
		},
		{
			name:      "empty topics",
			urls:      []string{"url1", "url2"},
			topics:    nil,
			groupID:   "one",
			expectErr: true,
		},
	}

	for _, tt := range testCases {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			consumer, err := NewConsumer(tt.urls, tt.topics, tt.groupID, tt.options...)

			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.Nil(t, err)
				require.NotNil(t, consumer, "consumerClient is nil")

				timeout, err := consumer.cfg.configMap.Get("session.timeout.ms", 0)
				require.Nil(t, err)
				require.Equal(t, int(tt.expectedTimeout.Milliseconds()), timeout)

				offset, err := consumer.cfg.configMap.Get("auto.offset.reset", string(OffsetNone))
				require.Nil(t, err)
				require.Equal(t, string(tt.expectedAutoOffsetResetPolicy), offset)

				require.Nil(t, consumer.Close())
			}
		})
	}
}

type mockConsumerClient struct{}

func (m mockConsumerClient) ReadMessage(_ time.Duration) (*kafka.Message, error) {
	return &kafka.Message{Value: []byte{1}}, nil
}

func (m mockConsumerClient) Close() error {
	return nil
}

func TestConsumerReadMessage(t *testing.T) {
	t.Parallel()

	consumer, err := NewConsumer(
		[]string{"url1", "url2"},
		[]string{"topic1", "topic2"},
		"group1",
	)
	require.Nil(t, err, "NewConsumer() unexpected error = %v", err)

	msg, err := consumer.ReadMessage()
	require.Error(t, err)
	require.Nil(t, msg)

	consumer.client = mockConsumerClient{}
	msg, err = consumer.ReadMessage()
	require.NoError(t, err)
	require.NotNil(t, msg)
}
