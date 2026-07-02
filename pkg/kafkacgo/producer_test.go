package kafkacgo

import (
	"errors"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
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
				WithFlushTimeout(time.Millisecond * 10),
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

				// This real (unconnectable) producer may or may not have
				// connection-error events queued at close time, so Close's
				// unflushed-events result is inherently racy here; Close
				// semantics are covered by the mock-based tests.
				_ = producer.Close()
			}
		})
	}
}

type mockProducerClient struct{}

func (m mockProducerClient) Produce(_ *kafka.Message, deliveryChan chan kafka.Event) error {
	deliveryChan <- &kafka.Message{} // report a successful delivery (nil TopicPartition.Error)

	return nil
}

func (m mockProducerClient) Flush(int) int { return 0 }

func (m mockProducerClient) Close() {}

type mockProducerClientError struct{}

func (m mockProducerClientError) Produce(_ *kafka.Message, _ chan kafka.Event) error {
	return errors.New("error Produce")
}

func (m mockProducerClientError) Flush(int) int { return 0 }

func (m mockProducerClientError) Close() {}

// fakeEvent is a kafka.Event that is not a *kafka.Message.
type fakeEvent struct{}

func (fakeEvent) String() string { return "fake" }

// flushMock is a producerClient whose Flush reports a configurable number of
// still-undelivered messages and records whether Close was called.
type flushMock struct {
	remaining int
	closed    *bool
}

func (f flushMock) Produce(_ *kafka.Message, _ chan kafka.Event) error { return nil }

func (f flushMock) Flush(int) int { return f.remaining }

func (f flushMock) Close() {
	if f.closed != nil {
		*f.closed = true
	}
}

func Test_Close(t *testing.T) {
	t.Parallel()

	producer, err := NewProducer([]string{"url"})
	require.NoError(t, err)
	require.NotNil(t, producer)

	// all messages flushed: no error and the client is closed
	closed := false
	producer.client = flushMock{remaining: 0, closed: &closed}
	require.NoError(t, producer.Close())
	require.True(t, closed)

	// unflushed events after the flush timeout: error and the client is still closed
	closed = false
	producer.client = flushMock{remaining: 7, closed: &closed}
	err = producer.Close()
	require.Error(t, err)
	require.ErrorContains(t, err, "7 unflushed events")
	require.True(t, closed)
}

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

func Test_Send_delivery_error(t *testing.T) {
	t.Parallel()

	producer, err := NewProducer([]string{"url"})
	require.NoError(t, err)

	producer.client = produceMock{
		produce: func(_ *kafka.Message, deliveryChan chan kafka.Event) error {
			deliveryChan <- &kafka.Message{
				TopicPartition: kafka.TopicPartition{Error: errors.New("delivery failed")},
			}

			return nil
		},
	}

	err = producer.Send("topic", []byte("payload"))
	require.Error(t, err)
}

func Test_Send_unexpected_event(t *testing.T) {
	t.Parallel()

	producer, err := NewProducer([]string{"url"})
	require.NoError(t, err)

	producer.client = produceMock{
		produce: func(_ *kafka.Message, deliveryChan chan kafka.Event) error {
			deliveryChan <- fakeEvent{}

			return nil
		},
	}

	err = producer.Send("topic", []byte("payload"))
	require.Error(t, err)
}

type produceMock struct {
	produce func(msg *kafka.Message, deliveryChan chan kafka.Event) error
	close   func()
}

func (p produceMock) Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error {
	return p.produce(msg, deliveryChan)
}

func (p produceMock) Flush(int) int { return 0 }

func (p produceMock) Close() {}

func TestSendData(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	cli, err := NewProducer([]string{"testurl"})
	require.NoError(t, err)
	require.NotNil(t, cli)

	cli.client = produceMock{
		produce: func(_ *kafka.Message, deliveryChan chan kafka.Event) error {
			deliveryChan <- &kafka.Message{}

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
