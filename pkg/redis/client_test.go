package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	libredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	srvOpts := &SrvOptions{
		Addr:     "test.redis.invalid:6379",
		Username: "test_user",
		Password: "test_password",
		DB:       0,
	}

	got, err := New(
		t.Context(),
		srvOpts,
		WithMessageEncodeFunc(nil),
	)

	require.Error(t, err)
	require.ErrorIs(t, err, ErrNilEncodeFunc)
	require.Nil(t, got)

	got, err = New(
		t.Context(),
		srvOpts,
		WithMessageDecodeFunc(nil),
	)

	require.Error(t, err)
	require.ErrorIs(t, err, ErrNilDecodeFunc)
	require.Nil(t, got)

	got, err = New(
		t.Context(),
		nil,
	)

	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidOptions)
	require.Nil(t, got)

	got, err = New(
		t.Context(),
		srvOpts,
		WithChannels("test_channel_1", ""),
	)

	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidChannelName)
	require.Nil(t, got)

	got, err = New(
		t.Context(),
		srvOpts,
	)

	require.NoError(t, err)
	require.NotNil(t, got)

	t.Cleanup(func() { _ = got.Close() })
}

func TestNew_withSubscription(t *testing.T) {
	t.Parallel()

	srvOpts := &SrvOptions{
		Addr:     "test.redis.invalid:6379",
		Username: "test_user",
		Password: "test_password",
		DB:       0,
	}

	got, err := New(
		t.Context(),
		srvOpts,
		WithChannels("test_channel_1", "test_channel_2"),
		WithChannelOptions(libredis.WithChannelSize(1)),
	)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.rpubsub)
	require.NotNil(t, got.subch)
	require.Equal(t, 1, cap(got.subch))

	t.Cleanup(func() { _ = got.Close() })
}

// TestNew_withInjectedClient_nilPubSub verifies that an injected client whose
// Subscribe returns a nil PubSub is reported as an error instead of a panic.
func TestNew_withInjectedClient_nilPubSub(t *testing.T) {
	t.Parallel()

	got, err := New(
		t.Context(),
		nil,
		WithRedisClient(redisClientMock{subscribeFn: func(_ context.Context, _ ...string) *libredis.PubSub {
			return nil
		}}),
		WithChannels("test_channel_1"),
	)

	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidOptions)
	require.Nil(t, got)
}

func TestNew_withInjectedClient(t *testing.T) {
	t.Parallel()

	// With an injected client no connection is dialed,
	// so no server options are required.
	cli, err := New(
		t.Context(),
		nil,
		WithRedisClient(redisClientMock{pingFn: func(_ context.Context) *libredis.StatusCmd {
			return libredis.NewStatusResult("PONG", nil)
		}}),
	)

	require.NoError(t, err)
	require.NotNil(t, cli)
	require.NoError(t, cli.HealthCheck(t.Context()))
}

func TestNew_withoutSubscription(t *testing.T) {
	t.Parallel()

	cli, err := New(
		t.Context(),
		nil,
		WithRedisClient(redisClientMock{}),
	)
	require.NoError(t, err)
	require.NotNil(t, cli)

	// No subscription was configured: no Pub/Sub resources are allocated.
	require.Nil(t, cli.rpubsub)
	require.Nil(t, cli.subch)

	// Receive must fail safely instead of blocking forever on a nil channel.
	ch, val, err := cli.Receive(t.Context())
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNoSubscription)
	require.Empty(t, ch)
	require.Empty(t, val)

	// Close must not panic when there is no Pub/Sub subscription.
	require.NoError(t, cli.Close())
}

type redisClientMock struct {
	closeFn     func() error
	delFn       func(ctx context.Context, keys ...string) *libredis.IntCmd
	getFn       func(ctx context.Context, key string) *libredis.StringCmd
	pingFn      func(ctx context.Context) *libredis.StatusCmd
	publishFn   func(ctx context.Context, channel string, message any) *libredis.IntCmd
	setFn       func(ctx context.Context, key string, value any, expiration time.Duration) *libredis.StatusCmd
	subscribeFn func(ctx context.Context, channels ...string) *libredis.PubSub
}

func (m redisClientMock) Close() error {
	if m.closeFn == nil {
		return nil
	}

	return m.closeFn()
}

func (m redisClientMock) Del(ctx context.Context, keys ...string) *libredis.IntCmd {
	return m.delFn(ctx, keys...)
}

func (m redisClientMock) Get(ctx context.Context, key string) *libredis.StringCmd {
	return m.getFn(ctx, key)
}

func (m redisClientMock) Ping(ctx context.Context) *libredis.StatusCmd {
	return m.pingFn(ctx)
}

func (m redisClientMock) Publish(ctx context.Context, channel string, message any) *libredis.IntCmd {
	return m.publishFn(ctx, channel, message)
}

func (m redisClientMock) Set(ctx context.Context, key string, value any, expiration time.Duration) *libredis.StatusCmd {
	return m.setFn(ctx, key, value, expiration)
}

func (m redisClientMock) Subscribe(ctx context.Context, channels ...string) *libredis.PubSub {
	return m.subscribeFn(ctx, channels...)
}

type redisPubSubMock struct {
	channelFn func(opts ...libredis.ChannelOption) <-chan *libredis.Message
	closeFn   func() error
}

func (m redisPubSubMock) Channel(opts ...libredis.ChannelOption) <-chan *libredis.Message {
	return m.channelFn(opts...)
}

func (m redisPubSubMock) Close() error {
	if m.closeFn == nil {
		return nil
	}

	return m.closeFn()
}

// requireErrorMatches asserts that err is non-nil and, when set, that it
// contains the wantMsg substring and matches the wantIs sentinel.
func requireErrorMatches(t *testing.T, err error, wantMsg string, wantIs error) {
	t.Helper()

	require.Error(t, err)

	if wantMsg != "" {
		require.ErrorContains(t, err, wantMsg)
	}

	if wantIs != nil {
		require.ErrorIs(t, err, wantIs)
	}
}

// newTestClient builds a Client on an injected mock, optionally wiring a
// mocked Pub/Sub subscription, without dialing any connection.
func newTestClient(t *testing.T, rclient RClient, rpubsub RPubSub) *Client {
	t.Helper()

	cli, err := New(t.Context(), nil, WithRedisClient(rclient))
	require.NoError(t, err)
	require.NotNil(t, cli)

	if rpubsub != nil {
		cli.rpubsub = rpubsub
		cli.subch = rpubsub.Channel()
	}

	t.Cleanup(func() { _ = cli.Close() })

	return cli
}

func TestClose(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		rClientMock RClient
		rPubSubMock RPubSub
		wantErr     bool
	}{
		{
			name: "success",
			rClientMock: redisClientMock{closeFn: func() error {
				return nil
			}},
			rPubSubMock: redisPubSubMock{closeFn: func() error {
				return nil
			}},
			wantErr: false,
		},
		{
			name: "error PubSub",
			rClientMock: redisClientMock{closeFn: func() error {
				return nil
			}},
			rPubSubMock: redisPubSubMock{closeFn: func() error {
				return errors.New("test error")
			}},
			wantErr: true,
		},
		{
			name: "error Client",
			rClientMock: redisClientMock{closeFn: func() error {
				return errors.New("test error")
			}},
			rPubSubMock: redisPubSubMock{closeFn: func() error {
				return nil
			}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cli, err := New(t.Context(), nil, WithRedisClient(tt.rClientMock))
			require.NoError(t, err)
			require.NotNil(t, cli)

			cli.rpubsub = tt.rPubSubMock

			err = cli.Close()
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

// TestClose_always_closes_client verifies that the Redis client is closed and
// its error reported even when closing the Pub/Sub subscription fails.
func TestClose_always_closes_client(t *testing.T) {
	t.Parallel()

	clientClosed := false

	cli := newTestClient(t,
		redisClientMock{closeFn: func() error {
			clientClosed = true

			return errors.New("client close error")
		}},
		redisPubSubMock{
			channelFn: func(_ ...libredis.ChannelOption) <-chan *libredis.Message {
				return make(chan *libredis.Message)
			},
			closeFn: func() error {
				return errors.New("pubsub close error")
			},
		},
	)

	err := cli.Close()
	require.Error(t, err)
	require.True(t, clientClosed, "rclient.Close must be attempted even when PubSub close fails")
	require.ErrorContains(t, err, "pubsub close error")
	require.ErrorContains(t, err, "client close error")
}

// TestClose_idempotent verifies that only the first Close call releases the
// resources and that subsequent calls return the result of the first.
func TestClose_idempotent(t *testing.T) {
	t.Parallel()

	closeCalls := 0

	cli := newTestClient(t,
		redisClientMock{closeFn: func() error {
			closeCalls++

			return errors.New("client close error")
		}},
		nil,
	)

	errFirst := cli.Close()
	require.Error(t, errFirst)

	errSecond := cli.Close()
	require.ErrorIs(t, errSecond, errFirst)
	require.Equal(t, 1, closeCalls, "rclient.Close must be called exactly once")
}

// TestClose_concurrentReceive verifies that closing the client while a
// Receive call is blocked unblocks it with ErrSubscriptionClosed, mirroring
// the real go-redis behavior where Close terminates the message channel.
func TestClose_concurrentReceive(t *testing.T) {
	t.Parallel()

	msgch := make(chan *libredis.Message)

	cli := newTestClient(t,
		redisClientMock{},
		redisPubSubMock{
			channelFn: func(_ ...libredis.ChannelOption) <-chan *libredis.Message {
				return msgch
			},
			closeFn: func() error {
				close(msgch)

				return nil
			},
		},
	)

	var (
		rerr error
		done = make(chan struct{})
	)

	go func() {
		defer close(done)

		_, _, rerr = cli.Receive(t.Context())
	}()

	require.NoError(t, cli.Close())

	<-done
	require.ErrorIs(t, rerr, ErrSubscriptionClosed)
}

func TestSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		rClientMock RClient
		wantErr     bool
	}{
		{
			name: "success",
			rClientMock: redisClientMock{setFn: func(_ context.Context, _ string, _ any, _ time.Duration) *libredis.StatusCmd {
				return libredis.NewStatusResult("", nil)
			}},
			wantErr: false,
		},
		{
			name: "error",
			rClientMock: redisClientMock{setFn: func(_ context.Context, _ string, _ any, _ time.Duration) *libredis.StatusCmd {
				return libredis.NewStatusResult("", errors.New("test error"))
			}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cli := newTestClient(t, tt.rClientMock, nil)

			err := cli.Set(t.Context(), "key_1", "value_1", time.Second)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		rClientMock RClient
		wantErr     bool
	}{
		{
			name: "success",
			rClientMock: redisClientMock{getFn: func(_ context.Context, _ string) *libredis.StringCmd {
				return libredis.NewStringResult("value_2", nil)
			}},
			wantErr: false,
		},
		{
			name: "error",
			rClientMock: redisClientMock{getFn: func(_ context.Context, _ string) *libredis.StringCmd {
				return libredis.NewStringResult("", errors.New("test error"))
			}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cli := newTestClient(t, tt.rClientMock, nil)

			var got string

			err := cli.Get(t.Context(), "key_2", &got)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, "value_2", got)
		})
	}
}

// TestGet_keyNotFound verifies that a missing key surfaces as ErrKeyNotFound
// and that the upstream go-redis Nil error is no longer in the chain.
func TestGet_keyNotFound(t *testing.T) {
	t.Parallel()

	cli := newTestClient(t,
		redisClientMock{getFn: func(_ context.Context, _ string) *libredis.StringCmd {
			return libredis.NewStringResult("", libredis.Nil)
		}},
		nil,
	)

	var got string

	err := cli.Get(t.Context(), "missing_key", &got)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrKeyNotFound)
	require.NotErrorIs(t, err, libredis.Nil)
	require.ErrorContains(t, err, "missing_key")
}

func TestDel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		rClientMock RClient
		wantErr     bool
	}{
		{
			name: "success",
			rClientMock: redisClientMock{delFn: func(_ context.Context, _ ...string) *libredis.IntCmd {
				return libredis.NewIntResult(0, nil)
			}},
			wantErr: false,
		},
		{
			name: "error",
			rClientMock: redisClientMock{delFn: func(_ context.Context, _ ...string) *libredis.IntCmd {
				return libredis.NewIntResult(0, errors.New("test error"))
			}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cli := newTestClient(t, tt.rClientMock, nil)

			err := cli.Del(t.Context(), "key_3")
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestSend(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		rClientMock RClient
		wantErr     bool
	}{
		{
			name: "success",
			rClientMock: redisClientMock{publishFn: func(_ context.Context, _ string, _ any) *libredis.IntCmd {
				return libredis.NewIntResult(0, nil)
			}},
			wantErr: false,
		},
		{
			name: "error",
			rClientMock: redisClientMock{publishFn: func(_ context.Context, _ string, _ any) *libredis.IntCmd {
				return libredis.NewIntResult(0, errors.New("test error"))
			}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cli := newTestClient(t, tt.rClientMock, nil)

			err := cli.Send(t.Context(), "channel_1", "message_1")
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestSetData(t *testing.T) {
	t.Parallel()

	cli := newTestClient(t,
		redisClientMock{setFn: func(_ context.Context, _ string, _ any, _ time.Duration) *libredis.StatusCmd {
			return libredis.NewStatusResult("", nil)
		}},
		nil,
	)

	type TestData struct {
		Alpha string
		Beta  int
	}

	err := cli.SetData(t.Context(), "key_4", TestData{Alpha: "abc123", Beta: -567}, time.Second)
	require.NoError(t, err)

	err = cli.SetData(t.Context(), "key_5", nil, time.Second)
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot encode data for key key_5")
}

func TestGetData(t *testing.T) {
	t.Parallel()

	type TestData struct {
		Alpha string
		Beta  int
	}

	tests := []struct {
		name        string
		rClientMock RClient
		wantErr     bool
		wantErrMsg  string
		wantErrIs   error
	}{
		{
			name: "success",
			rClientMock: redisClientMock{getFn: func(_ context.Context, _ string) *libredis.StringCmd {
				return libredis.NewStringResult("Kf+BAwEBCFRlc3REYXRhAf+CAAECAQVBbHBoYQEMAAEEQmV0YQEEAAAAD/+CAQZhYmMxMjMB/gLtAA==", nil)
			}},
			wantErr: false,
		},
		{
			name: "error corrupted value",
			rClientMock: redisClientMock{getFn: func(_ context.Context, _ string) *libredis.StringCmd {
				return libredis.NewStringResult("INVALID", nil)
			}},
			wantErr:    true,
			wantErrMsg: "cannot decode data for key key_7",
		},
		{
			name: "error key not found",
			rClientMock: redisClientMock{getFn: func(_ context.Context, _ string) *libredis.StringCmd {
				return libredis.NewStringResult("", libredis.Nil)
			}},
			wantErr:   true,
			wantErrIs: ErrKeyNotFound,
		},
		{
			name: "error",
			rClientMock: redisClientMock{getFn: func(_ context.Context, _ string) *libredis.StringCmd {
				return libredis.NewStringResult("", errors.New("test error"))
			}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cli := newTestClient(t, tt.rClientMock, nil)

			var data TestData

			err := cli.GetData(t.Context(), "key_7", &data)
			if tt.wantErr {
				requireErrorMatches(t, err, tt.wantErrMsg, tt.wantErrIs)
				return
			}

			require.NoError(t, err)
			require.Equal(t, "abc123", data.Alpha)
			require.Equal(t, -375, data.Beta)
		})
	}
}

func TestSendData(t *testing.T) {
	t.Parallel()

	cli := newTestClient(t,
		redisClientMock{publishFn: func(_ context.Context, _ string, _ any) *libredis.IntCmd {
			return libredis.NewIntResult(0, nil)
		}},
		nil,
	)

	type TestData struct {
		Alpha string
		Beta  int
	}

	err := cli.SendData(t.Context(), "channel_2", TestData{Alpha: "abc345", Beta: -678})
	require.NoError(t, err)

	err = cli.SendData(t.Context(), "channel_3", nil)
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot encode data for channel_3 channel")
}

func TestReceive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		rPubSubMock RPubSub
		ctxTimeout  time.Duration
		wantErr     bool
		wantErrIs   error
	}{
		{
			name: "success",
			rPubSubMock: redisPubSubMock{
				channelFn: func(_ ...libredis.ChannelOption) <-chan *libredis.Message {
					ch := make(chan *libredis.Message)

					go func() {
						ch <- &libredis.Message{
							Channel: "channel_4",
							Payload: "message_4",
						}
					}()

					return ch
				},
			},
			wantErr: false,
		},
		{
			name: "success skips nil message",
			rPubSubMock: redisPubSubMock{
				channelFn: func(_ ...libredis.ChannelOption) <-chan *libredis.Message {
					ch := make(chan *libredis.Message)

					go func() {
						ch <- nil

						ch <- &libredis.Message{
							Channel: "channel_4",
							Payload: "message_4",
						}
					}()

					return ch
				},
			},
			wantErr: false,
		},
		{
			name: "error closed channel",
			rPubSubMock: redisPubSubMock{
				channelFn: func(_ ...libredis.ChannelOption) <-chan *libredis.Message {
					ch := make(chan *libredis.Message)

					go func() {
						close(ch)
					}()

					return ch
				},
			},
			wantErr:   true,
			wantErrIs: ErrSubscriptionClosed,
		},
		{
			name: "error context timeout",
			rPubSubMock: redisPubSubMock{
				channelFn: func(_ ...libredis.ChannelOption) <-chan *libredis.Message {
					return make(chan *libredis.Message)
				},
			},
			ctxTimeout: 1 * time.Millisecond,
			wantErr:    true,
			wantErrIs:  context.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cli := newTestClient(t, redisClientMock{}, tt.rPubSubMock)

			ctx := t.Context()

			if tt.ctxTimeout > 0 {
				cctx, cancel := context.WithTimeout(ctx, tt.ctxTimeout)
				ctx = cctx

				defer cancel()
			}

			ch, val, err := cli.Receive(ctx)
			if tt.wantErr {
				requireErrorMatches(t, err, "", tt.wantErrIs)
				return
			}

			require.NoError(t, err)
			require.Equal(t, "channel_4", ch)
			require.Equal(t, "message_4", val)
		})
	}
}

func TestReceiveData(t *testing.T) {
	t.Parallel()

	type TestData struct {
		Alpha string
		Beta  int
	}

	tests := []struct {
		name        string
		rPubSubMock RPubSub
		wantErr     bool
		wantErrMsg  string
	}{
		{
			name: "success",
			rPubSubMock: redisPubSubMock{
				channelFn: func(_ ...libredis.ChannelOption) <-chan *libredis.Message {
					ch := make(chan *libredis.Message)

					go func() {
						ch <- &libredis.Message{
							Channel: "channel_5",
							Payload: "Kf+BAwEBCFRlc3REYXRhAf+CAAECAQVBbHBoYQEMAAEEQmV0YQEEAAAAD/+CAQZhYmMxMjMB/gLtAA==",
						}
					}()

					return ch
				},
			},
			wantErr: false,
		},
		{
			name: "error corrupted value",
			rPubSubMock: redisPubSubMock{
				channelFn: func(_ ...libredis.ChannelOption) <-chan *libredis.Message {
					ch := make(chan *libredis.Message)

					go func() {
						ch <- &libredis.Message{
							Channel: "channel_5",
							Payload: "INVALID",
						}
					}()

					return ch
				},
			},
			wantErr:    true,
			wantErrMsg: "cannot decode data from channel_5 channel",
		},
		{
			name: "error closed channel",
			rPubSubMock: redisPubSubMock{
				channelFn: func(_ ...libredis.ChannelOption) <-chan *libredis.Message {
					ch := make(chan *libredis.Message)

					go func() {
						close(ch)
					}()

					return ch
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cli := newTestClient(t, redisClientMock{}, tt.rPubSubMock)

			var data TestData

			ch, err := cli.ReceiveData(t.Context(), &data)
			if tt.wantErr {
				requireErrorMatches(t, err, tt.wantErrMsg, nil)

				// On any error, including a decode failure,
				// the returned channel name is empty.
				require.Empty(t, ch)

				return
			}

			require.NoError(t, err)
			require.Equal(t, "channel_5", ch)
			require.Equal(t, "abc123", data.Alpha)
			require.Equal(t, -375, data.Beta)
		})
	}
}

func TestHealthCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		rClientMock RClient
		wantErr     bool
	}{
		{
			name: "success",
			rClientMock: redisClientMock{pingFn: func(_ context.Context) *libredis.StatusCmd {
				return libredis.NewStatusResult("", nil)
			}},
			wantErr: false,
		},
		{
			name: "error",
			rClientMock: redisClientMock{pingFn: func(_ context.Context) *libredis.StatusCmd {
				return libredis.NewStatusResult("", errors.New("test error"))
			}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cli := newTestClient(t, tt.rClientMock, nil)

			err := cli.HealthCheck(t.Context())
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
