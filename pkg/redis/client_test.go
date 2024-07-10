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
		context.TODO(),
		srvOpts,
		WithMessageEncodeFunc(nil),
	)

	require.Error(t, err)
	require.Nil(t, got)

	got, err = New(
		context.TODO(),
		srvOpts,
	)

	require.NoError(t, err)
	require.NotNil(t, got)
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
	return m.closeFn()
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

			srvOpts := &SrvOptions{
				Addr:     "test.redis.invalid:6379",
				Username: "test_user",
				Password: "test_password",
				DB:       0,
			}

			ctx := context.TODO()
			cli, err := New(ctx, srvOpts)
			require.NoError(t, err)
			require.NotNil(t, cli)

			cli.rclient = tt.rClientMock
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

			srvOpts := &SrvOptions{
				Addr:     "test.redis.invalid:6379",
				Username: "test_user",
				Password: "test_password",
				DB:       0,
			}

			ctx := context.TODO()
			cli, err := New(ctx, srvOpts)
			require.NoError(t, err)
			require.NotNil(t, cli)

			cli.rclient = tt.rClientMock

			err = cli.Set(ctx, "key_1", "value_1", time.Second)
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

			srvOpts := &SrvOptions{
				Addr:     "test.redis.invalid:6379",
				Username: "test_user",
				Password: "test_password",
				DB:       0,
			}

			ctx := context.TODO()
			cli, err := New(ctx, srvOpts)
			require.NoError(t, err)
			require.NotNil(t, cli)

			cli.rclient = tt.rClientMock

			var got string

			err = cli.Get(ctx, "key_2", &got)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, "value_2", got)
		})
	}
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

			srvOpts := &SrvOptions{
				Addr:     "test.redis.invalid:6379",
				Username: "test_user",
				Password: "test_password",
				DB:       0,
			}

			ctx := context.TODO()
			cli, err := New(ctx, srvOpts)
			require.NoError(t, err)
			require.NotNil(t, cli)

			cli.rclient = tt.rClientMock

			err = cli.Del(ctx, "key_3")
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

			srvOpts := &SrvOptions{
				Addr:     "test.redis.invalid:6379",
				Username: "test_user",
				Password: "test_password",
				DB:       0,
			}

			ctx := context.TODO()
			cli, err := New(ctx, srvOpts)
			require.NoError(t, err)
			require.NotNil(t, cli)

			cli.rclient = tt.rClientMock

			err = cli.Send(ctx, "channel_1", "message_1")
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

	srvOpts := &SrvOptions{
		Addr:     "test.redis.invalid:6379",
		Username: "test_user",
		Password: "test_password",
		DB:       0,
	}

	ctx := context.TODO()
	cli, err := New(ctx, srvOpts)
	require.NoError(t, err)
	require.NotNil(t, cli)

	cli.rclient = redisClientMock{setFn: func(_ context.Context, _ string, _ any, _ time.Duration) *libredis.StatusCmd {
		return libredis.NewStatusResult("", nil)
	}}

	type TestData struct {
		Alpha string
		Beta  int
	}

	err = cli.SetData(ctx, "key_4", TestData{Alpha: "abc123", Beta: -567}, time.Second)
	require.NoError(t, err)

	err = cli.SetData(ctx, "key_5", nil, time.Second)
	require.Error(t, err)
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
			wantErr: true,
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

			srvOpts := &SrvOptions{
				Addr:     "test.redis.invalid:6379",
				Username: "test_user",
				Password: "test_password",
				DB:       0,
			}

			ctx := context.TODO()
			cli, err := New(ctx, srvOpts)
			require.NoError(t, err)
			require.NotNil(t, cli)

			cli.rclient = tt.rClientMock

			var data TestData

			err = cli.GetData(ctx, "key_7", &data)
			if tt.wantErr {
				require.Error(t, err)
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

	srvOpts := &SrvOptions{
		Addr:     "test.redis.invalid:6379",
		Username: "test_user",
		Password: "test_password",
		DB:       0,
	}

	ctx := context.TODO()
	cli, err := New(ctx, srvOpts)
	require.NoError(t, err)
	require.NotNil(t, cli)

	cli.rclient = redisClientMock{publishFn: func(_ context.Context, _ string, _ any) *libredis.IntCmd {
		return libredis.NewIntResult(0, nil)
	}}

	type TestData struct {
		Alpha string
		Beta  int
	}

	err = cli.SendData(ctx, "channel_2", TestData{Alpha: "abc345", Beta: -678})
	require.NoError(t, err)

	err = cli.SendData(ctx, "channel_3", nil)
	require.Error(t, err)
}

func TestReceive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		rPubSubMock RPubSub
		ctxTimeout  time.Duration
		wantErr     bool
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
		{
			name: "error context timeout",
			rPubSubMock: redisPubSubMock{
				channelFn: func(_ ...libredis.ChannelOption) <-chan *libredis.Message {
					return make(chan *libredis.Message)
				},
			},
			ctxTimeout: 1 * time.Millisecond,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srvOpts := &SrvOptions{
				Addr:     "test.redis.invalid:6379",
				Username: "test_user",
				Password: "test_password",
				DB:       0,
			}

			ctx := context.TODO()
			cli, err := New(ctx, srvOpts)
			require.NoError(t, err)
			require.NotNil(t, cli)

			cli.rpubsub = tt.rPubSubMock
			cli.subch = cli.rpubsub.Channel()

			if tt.ctxTimeout > 0 {
				cctx, cancel := context.WithTimeout(ctx, tt.ctxTimeout)
				ctx = cctx
				defer cancel()
			}

			ch, val, err := cli.Receive(ctx)
			if tt.wantErr {
				require.Error(t, err)
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
			wantErr: true,
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

			srvOpts := &SrvOptions{
				Addr:     "test.redis.invalid:6379",
				Username: "test_user",
				Password: "test_password",
				DB:       0,
			}

			ctx := context.TODO()
			cli, err := New(ctx, srvOpts)
			require.NoError(t, err)
			require.NotNil(t, cli)

			cli.rpubsub = tt.rPubSubMock
			cli.subch = cli.rpubsub.Channel()

			var data TestData

			ch, err := cli.ReceiveData(ctx, &data)
			if tt.wantErr {
				require.Error(t, err)
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

			srvOpts := &SrvOptions{
				Addr:     "test.redis.invalid:6379",
				Username: "test_user",
				Password: "test_password",
				DB:       0,
			}

			ctx := context.TODO()
			cli, err := New(ctx, srvOpts)
			require.NoError(t, err)
			require.NotNil(t, cli)

			cli.rclient = tt.rClientMock

			err = cli.HealthCheck(ctx)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
