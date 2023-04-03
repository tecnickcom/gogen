package kafka

import (
	"context"
	"fmt"
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
	}

	for _, tt := range testCases {
		tt := tt

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
	return fmt.Errorf("error WriteMessages")
}

func (m mockProducerClientError) Close() error {
	return fmt.Errorf("error Close")
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
