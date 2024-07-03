package kafka

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

func Test_NewProducer(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                  string
		urls                  []string
		topic                 string
		options               []Option
		expTimeout            time.Duration
		expProduceChannelSize int
		wantErr               bool
	}{
		{
			name: "success",
			urls: []string{"url1", "url2"},
			options: []Option{
				WithSessionTimeout(time.Millisecond * 17),
			},
			expTimeout: time.Millisecond * 17,
		},
		{
			name: "missing encoding function",
			urls: []string{"url1", "url2"},
			options: []Option{
				WithMessageEncodeFunc(nil),
			},
			expTimeout: time.Millisecond * 17,
			wantErr:    true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			producer, err := NewProducer(tt.urls, tt.topic, tt.options...)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, producer)
			} else {
				require.NoError(t, err)
				require.NotNil(t, producer)
				require.Equal(t, tt.expTimeout, producer.cfg.sessionTimeout)

				err := producer.Close()
				require.NoError(t, err)
			}
		})
	}
}

type mockProducerClient struct{}

func (m mockProducerClient) WriteMessages(_ context.Context, _ ...kafka.Message) error {
	return nil
}

func (m mockProducerClient) Close() error {
	return nil
}

type mockProducerClientError struct{}

func (m mockProducerClientError) WriteMessages(_ context.Context, _ ...kafka.Message) error {
	return errors.New("error WriteMessages")
}

func (m mockProducerClientError) Close() error {
	return errors.New("error Close")
}

func TestSendError(t *testing.T) {
	t.Parallel()

	producer, err := NewProducer([]string{"url"}, "test")

	require.NoError(t, err)
	require.NotNil(t, producer)

	producer.client = &mockProducerClient{}
	err = producer.Send(context.TODO(), nil)
	require.NoError(t, err)

	producer.client = &mockProducerClientError{}
	err = producer.Send(context.TODO(), nil)
	require.Error(t, err)

	err = producer.Close()
	require.Error(t, err)
}

type produceMock struct {
	writeMessages func(ctx context.Context, msg ...kafka.Message) error
	close         func() error
}

func (p produceMock) WriteMessages(ctx context.Context, msg ...kafka.Message) error {
	return p.writeMessages(ctx, msg...)
}

func (p produceMock) Close() error {
	return p.close()
}

func TestSendData(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	cli, err := NewProducer([]string{"testurl"}, "")
	require.NoError(t, err)
	require.NotNil(t, cli)

	cli.client = produceMock{
		writeMessages: func(_ context.Context, _ ...kafka.Message) error {
			return nil
		},
		close: func() error {
			return nil
		},
	}

	type TestData struct {
		Alpha string
		Beta  int
	}

	err = cli.SendData(ctx, TestData{Alpha: "abc345", Beta: -678})
	require.NoError(t, err)

	err = cli.SendData(ctx, nil)
	require.Error(t, err)
}
