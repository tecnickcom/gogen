package kafkacgo

import (
	"fmt"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/stretchr/testify/require"
)

func Test_NewConsumer(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                     string
		urls                     []string
		topics                   []string
		groupID                  string
		options                  []Option
		expTimeout               time.Duration
		expAutoOffsetResetPolicy Offset
		wantErr                  bool
	}{
		{
			name:    "success",
			urls:    []string{"url1", "url2"},
			topics:  []string{"topic1", "topic2"},
			groupID: "one",
			options: []Option{
				WithSessionTimeout(time.Millisecond * 13),
				WithAutoOffsetResetPolicy(OffsetLatest),
			},
			expTimeout:               time.Millisecond * 13,
			expAutoOffsetResetPolicy: OffsetLatest,
			wantErr:                  false,
		},
		{
			name:    "bad offset",
			urls:    []string{"url1", "url2"},
			topics:  []string{"topic1", "topic2"},
			groupID: "one",
			options: []Option{
				WithAutoOffsetResetPolicy("bad offset"),
			},
			wantErr: true,
		},
		{
			name:    "empty topics",
			urls:    []string{"url1", "url2"},
			topics:  nil,
			groupID: "one",
			wantErr: true,
		},
	}

	for _, tt := range testCases {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			consumer, err := NewConsumer(tt.urls, tt.topics, tt.groupID, tt.options...)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, consumer)
			} else {
				require.NoError(t, err)
				require.NotNil(t, consumer)

				timeout, err := consumer.cfg.configMap.Get("session.timeout.ms", 0)
				require.Nil(t, err)
				require.Equal(t, int(tt.expTimeout.Milliseconds()), timeout)

				offset, err := consumer.cfg.configMap.Get("auto.offset.reset", string(OffsetNone))
				require.Nil(t, err)
				require.Equal(t, string(tt.expAutoOffsetResetPolicy), offset)

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

type mockConsumerClientError struct{}

func (m mockConsumerClientError) ReadMessage(_ time.Duration) (*kafka.Message, error) {
	return nil, fmt.Errorf("error ReadMessage")
}

func (m mockConsumerClientError) Close() error {
	return fmt.Errorf("error Close")
}

func Test_Receive(t *testing.T) {
	t.Parallel()

	consumer, err := NewConsumer(
		[]string{"url1", "url2"},
		[]string{"topic1", "topic2"},
		"group1",
	)

	require.NoError(t, err)
	require.NotNil(t, consumer)

	consumer.client = mockConsumerClient{}
	msg, err := consumer.Receive()
	require.NoError(t, err)
	require.NotNil(t, msg)

	consumer.client = mockConsumerClientError{}
	msg, err = consumer.Receive()
	require.Error(t, err)
	require.Nil(t, msg)

	err = consumer.Close()
	require.Error(t, err)
}
