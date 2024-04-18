package kafkacgo

import (
	"context"
	"errors"
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
		{
			name:    "missing decoding function",
			urls:    []string{"url1", "url2"},
			topics:  []string{"topic1", "topic2"},
			groupID: "four",
			options: []Option{
				WithMessageDecodeFunc(nil),
			},
			expTimeout:               time.Millisecond * 17,
			expAutoOffsetResetPolicy: OffsetLatest,
			wantErr:                  true,
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
				require.NoError(t, err)
				require.Equal(t, int(tt.expTimeout.Milliseconds()), timeout)

				offset, err := consumer.cfg.configMap.Get("auto.offset.reset", string(OffsetNone))
				require.NoError(t, err)
				require.Equal(t, string(tt.expAutoOffsetResetPolicy), offset)

				require.NoError(t, consumer.Close())
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
	return nil, errors.New("error ReadMessage")
}

func (m mockConsumerClientError) Close() error {
	return errors.New("error Close")
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

type consumerMock struct {
	readMessage func(duration time.Duration) (*kafka.Message, error)
	close       func() error
}

func (c consumerMock) ReadMessage(duration time.Duration) (*kafka.Message, error) {
	return c.readMessage(duration)
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
				readMessage: func(_ time.Duration) (*kafka.Message, error) {
					return &kafka.Message{
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
				readMessage: func(_ time.Duration) (*kafka.Message, error) {
					return &kafka.Message{
						Value: []byte{},
					}, nil
				},
				close: func() error { return nil },
			},
			wantErr: true,
		},
		{
			name: "error",
			mock: consumerMock{
				readMessage: func(_ time.Duration) (*kafka.Message, error) {
					return &kafka.Message{}, errors.New("error")
				},
				close: func() error { return nil },
			},
			wantErr: true,
		},
		{
			name: "invalid message",
			mock: consumerMock{
				readMessage: func(_ time.Duration) (*kafka.Message, error) {
					return &kafka.Message{
						Value: []byte("你好世界"),
					}, nil
				},
				close: func() error { return nil },
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.TODO()
			cli, err := NewConsumer([]string{"url1", "url2"}, []string{"topic"}, "groupID")
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
