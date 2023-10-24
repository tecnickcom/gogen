package kafkacgo

import (
	"fmt"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/stretchr/testify/require"
)

func Test_NewProducer(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                  string
		urls                  []string
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
				WithProduceChannelSize(1_000),
			},
			expTimeout:            time.Millisecond * 17,
			expProduceChannelSize: 1_000,
		},
		{
			name: "bad param",
			urls: []string{"url1", "url2"},
			options: []Option{
				WithSessionTimeout(time.Millisecond * 15),
				WithProduceChannelSize(1_000),
				WithConfigParameter("badkey", 99),
			},
			expTimeout:            time.Millisecond * 15,
			expProduceChannelSize: 1_000,
			wantErr:               true,
		},
	}

	for _, tt := range testCases {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			producer, err := NewProducer(tt.urls, tt.options...)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, producer)
			} else {
				require.NoError(t, err)
				require.NotNil(t, producer)

				timeout, err := producer.cfg.configMap.Get("session.timeout.ms", 0)
				require.NoError(t, err)
				require.Equal(t, int(tt.expTimeout.Milliseconds()), timeout)

				offset, err := producer.cfg.configMap.Get("go.produce.channel.size", 0)
				require.NoError(t, err)
				require.Equal(t, tt.expProduceChannelSize, offset)

				producer.Close()
			}
		})
	}
}

type mockProducerClient struct{}

func (m mockProducerClient) Produce(_ *kafka.Message, _ chan kafka.Event) error {
	return nil
}

func (m mockProducerClient) Close() {}

type mockProducerClientError struct{}

func (m mockProducerClientError) Produce(_ *kafka.Message, _ chan kafka.Event) error {
	return fmt.Errorf("error Produce")
}

func (m mockProducerClientError) Close() {}

func Test_Send(t *testing.T) {
	t.Parallel()

	producer, err := NewProducer([]string{"url"})

	require.NoError(t, err)
	require.NotNil(t, producer)

	producer.client = mockProducerClient{}
	err = producer.Send("", nil)
	require.NoError(t, err)

	producer.client = mockProducerClientError{}
	err = producer.Send("", nil)
	require.Error(t, err)
}
