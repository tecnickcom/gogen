package kafkacgo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
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

func (m mockConsumerClient) SubscribeTopics(_ []string, _ kafka.RebalanceCb) error {
	return nil
}

func (m mockConsumerClient) ReadMessage(_ time.Duration) (*kafka.Message, error) {
	return &kafka.Message{Value: []byte{1}}, nil
}

func (m mockConsumerClient) CommitMessage(_ *kafka.Message) ([]kafka.TopicPartition, error) {
	return nil, nil
}

func (m mockConsumerClient) StoreMessage(_ *kafka.Message) ([]kafka.TopicPartition, error) {
	return nil, nil
}

func (m mockConsumerClient) Close() error {
	return nil
}

type mockConsumerClientError struct{}

func (m mockConsumerClientError) SubscribeTopics(_ []string, _ kafka.RebalanceCb) error {
	return errors.New("error SubscribeTopics")
}

func (m mockConsumerClientError) ReadMessage(_ time.Duration) (*kafka.Message, error) {
	return nil, errors.New("error ReadMessage")
}

func (m mockConsumerClientError) CommitMessage(_ *kafka.Message) ([]kafka.TopicPartition, error) {
	return nil, errors.New("error CommitMessage")
}

func (m mockConsumerClientError) StoreMessage(_ *kafka.Message) ([]kafka.TopicPartition, error) {
	return nil, errors.New("error StoreMessage")
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

func Test_ReceiveCtx(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		consumer := newTestConsumer(t)
		consumer.client = consumerMock{
			readMessage: func(_ time.Duration) (*kafka.Message, error) {
				return &kafka.Message{Value: []byte{1, 2, 3}}, nil
			},
			close: func() error { return nil },
		}

		msg, err := consumer.ReceiveCtx(t.Context())
		require.NoError(t, err)
		require.Equal(t, []byte{1, 2, 3}, msg)
	})

	t.Run("retries on timeout then succeeds", func(t *testing.T) {
		t.Parallel()

		consumer := newTestConsumer(t)

		var calls int

		consumer.client = consumerMock{
			readMessage: func(_ time.Duration) (*kafka.Message, error) {
				calls++
				if calls < 3 {
					return nil, kafka.NewError(kafka.ErrTimedOut, "timed out", false)
				}

				return &kafka.Message{Value: []byte{9}}, nil
			},
			close: func() error { return nil },
		}

		msg, err := consumer.ReceiveCtx(t.Context())
		require.NoError(t, err)
		require.Equal(t, []byte{9}, msg)
		require.Equal(t, 3, calls)
	})

	t.Run("returns non-timeout error", func(t *testing.T) {
		t.Parallel()

		consumer := newTestConsumer(t)
		consumer.client = consumerMock{
			readMessage: func(_ time.Duration) (*kafka.Message, error) {
				return nil, errors.New("boom")
			},
			close: func() error { return nil },
		}

		msg, err := consumer.ReceiveCtx(t.Context())
		require.Error(t, err)
		require.Nil(t, msg)
	})

	t.Run("returns when context already canceled", func(t *testing.T) {
		t.Parallel()

		consumer := newTestConsumer(t)
		consumer.client = consumerMock{
			readMessage: func(_ time.Duration) (*kafka.Message, error) {
				return &kafka.Message{Value: []byte{1}}, nil
			},
			close: func() error { return nil },
		}

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		msg, err := consumer.ReceiveCtx(ctx)
		require.Error(t, err)
		require.ErrorIs(t, err, context.Canceled)
		require.Nil(t, msg)
	})

	t.Run("returns when context canceled during timeout polling", func(t *testing.T) {
		t.Parallel()

		consumer := newTestConsumer(t)

		ctx, cancel := context.WithCancel(t.Context())
		consumer.client = consumerMock{
			readMessage: func(_ time.Duration) (*kafka.Message, error) {
				cancel() // cancel after the first read so the loop exits on the ctx check

				return nil, kafka.NewError(kafka.ErrTimedOut, "timed out", false)
			},
			close: func() error { return nil },
		}

		msg, err := consumer.ReceiveCtx(ctx)
		require.Error(t, err)
		require.ErrorIs(t, err, context.Canceled)
		require.Nil(t, msg)
	})
}

func Test_newConsumer_subscribe_error_closes_client(t *testing.T) {
	t.Parallel()

	t.Run("close succeeds", func(t *testing.T) {
		t.Parallel()

		closed := false
		mock := consumerMock{
			subscribeTopics: func(_ []string, _ kafka.RebalanceCb) error {
				return errors.New("error SubscribeTopics")
			},
			close: func() error {
				closed = true

				return nil
			},
		}

		consumer, err := newConsumer(defaultConfig(), mock, []string{"topic1"})
		require.Error(t, err)
		require.Nil(t, consumer)
		require.ErrorContains(t, err, "failed to subscribe kafka topic")
		require.True(t, closed, "the consumer client must be closed when SubscribeTopics fails")
	})

	t.Run("close error is joined", func(t *testing.T) {
		t.Parallel()

		mock := consumerMock{
			subscribeTopics: func(_ []string, _ kafka.RebalanceCb) error {
				return errors.New("error SubscribeTopics")
			},
			close: func() error {
				return errors.New("error Close")
			},
		}

		consumer, err := newConsumer(defaultConfig(), mock, []string{"topic1"})
		require.Error(t, err)
		require.Nil(t, consumer)
		require.ErrorContains(t, err, "failed to subscribe kafka topic")
		require.ErrorContains(t, err, "error Close")
	})
}

func Test_ReceiveMessage(t *testing.T) {
	t.Parallel()

	consumer := newTestConsumer(t)

	consumer.client = mockConsumerClient{}
	msg, err := consumer.ReceiveMessage(t.Context())
	require.NoError(t, err)
	require.NotNil(t, msg)
	require.Equal(t, []byte{1}, msg.Value)

	consumer.client = mockConsumerClientError{}
	msg, err = consumer.ReceiveMessage(t.Context())
	require.Error(t, err)
	require.Nil(t, msg)
}

func Test_CommitMessage(t *testing.T) {
	t.Parallel()

	consumer := newTestConsumer(t)
	msg := &kafka.Message{Value: []byte{1}}

	var committed *kafka.Message

	consumer.client = consumerMock{
		commitMessage: func(m *kafka.Message) ([]kafka.TopicPartition, error) {
			committed = m

			return nil, nil
		},
		close: func() error { return nil },
	}

	err := consumer.CommitMessage(msg)
	require.NoError(t, err)
	require.Same(t, msg, committed)

	consumer.client = mockConsumerClientError{}
	err = consumer.CommitMessage(msg)
	require.Error(t, err)
}

func Test_StoreMessage(t *testing.T) {
	t.Parallel()

	consumer := newTestConsumer(t)
	msg := &kafka.Message{Value: []byte{1}}

	var stored *kafka.Message

	consumer.client = consumerMock{
		storeMessage: func(m *kafka.Message) ([]kafka.TopicPartition, error) {
			stored = m

			return nil, nil
		},
		close: func() error { return nil },
	}

	err := consumer.StoreMessage(msg)
	require.NoError(t, err)
	require.Same(t, msg, stored)

	consumer.client = mockConsumerClientError{}
	err = consumer.StoreMessage(msg)
	require.Error(t, err)
}

func newTestConsumer(t *testing.T) *Consumer {
	t.Helper()

	consumer, err := NewConsumer([]string{"url1", "url2"}, []string{"topic1"}, "group1")
	require.NoError(t, err)
	require.NotNil(t, consumer)

	return consumer
}

type consumerMock struct {
	subscribeTopics func(topics []string, rebalanceCb kafka.RebalanceCb) error
	readMessage     func(duration time.Duration) (*kafka.Message, error)
	commitMessage   func(msg *kafka.Message) ([]kafka.TopicPartition, error)
	storeMessage    func(msg *kafka.Message) ([]kafka.TopicPartition, error)
	close           func() error
}

func (c consumerMock) SubscribeTopics(topics []string, rebalanceCb kafka.RebalanceCb) error {
	return c.subscribeTopics(topics, rebalanceCb)
}

func (c consumerMock) ReadMessage(duration time.Duration) (*kafka.Message, error) {
	return c.readMessage(duration)
}

func (c consumerMock) CommitMessage(msg *kafka.Message) ([]kafka.TopicPartition, error) {
	return c.commitMessage(msg)
}

func (c consumerMock) StoreMessage(msg *kafka.Message) ([]kafka.TopicPartition, error) {
	return c.storeMessage(msg)
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
