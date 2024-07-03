package kafkacgo

import (
	"context"
	"errors"
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
		{
			name: "missing encoding function",
			urls: []string{"url1", "url2"},
			options: []Option{
				WithMessageEncodeFunc(nil),
			},
			expTimeout:            time.Millisecond * 17,
			expProduceChannelSize: 1_000,
			wantErr:               true,
		},
	}

	for _, tt := range testCases {
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
	return errors.New("error Produce")
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

type produceMock struct {
	produce func(msg *kafka.Message, deliveryChan chan kafka.Event) error
	close   func()
}

func (p produceMock) Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error {
	return p.produce(msg, deliveryChan)
}

func (p produceMock) Close() {}

func TestSendData(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	cli, err := NewProducer([]string{"testurl"})
	require.NoError(t, err)
	require.NotNil(t, cli)

	cli.client = produceMock{
		produce: func(_ *kafka.Message, _ chan kafka.Event) error {
			return nil
		},
		close: func() {},
	}

	type TestData struct {
		Alpha string
		Beta  int
	}

	err = cli.SendData(ctx, "topic1", TestData{Alpha: "abc345", Beta: -678})
	require.NoError(t, err)

	err = cli.SendData(ctx, "topic2", nil)
	require.Error(t, err)
}
