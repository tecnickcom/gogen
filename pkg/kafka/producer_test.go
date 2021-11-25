package kafka

import (
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/stretchr/testify/require"
)

func TestProducer(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                       string
		urls                       []string
		options                    []Option
		expectedTimeout            time.Duration
		expectedProduceChannelSize int
		expectErr                  bool
	}{
		{
			name: "success",
			urls: []string{"url1", "url2"},
			options: []Option{
				WithSessionTimeout(time.Second * 10),
				WithProduceChannelSize(1_000),
			},
			expectedTimeout:            time.Second * 10,
			expectedProduceChannelSize: 1_000,
		},
		{
			name: "bad param",
			urls: []string{"url1", "url2"},
			options: []Option{
				WithSessionTimeout(time.Second * 10),
				WithProduceChannelSize(1_000),
				WithConfigParameter("badkey", 99),
			},
			expectedTimeout:            time.Second * 10,
			expectedProduceChannelSize: 1_000,
			expectErr:                  true,
		},
	}

	for _, tt := range testCases {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			producer, err := NewProducer(tt.urls, tt.options...)

			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NotNil(t, producer)
				require.Nil(t, err)

				timeout, err := producer.cfg.configMap.Get("session.timeout.ms", 0)
				require.Nil(t, err)
				require.Equal(t, int(tt.expectedTimeout.Milliseconds()), timeout)

				offset, err := producer.cfg.configMap.Get("go.produce.channel.size", 0)
				require.Nil(t, err)
				require.Equal(t, tt.expectedProduceChannelSize, offset)

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

func TestProduceMessageError(t *testing.T) {
	t.Parallel()

	producer, err := NewProducer([]string{"url"})
	require.Nil(t, err, "NewProducer() unexpected error = %v", err)

	err = producer.ProduceMessage("", nil)
	require.Error(t, err)

	producer.client = mockProducerClient{}
	err = producer.ProduceMessage("", nil)
	require.NoError(t, err)
}
