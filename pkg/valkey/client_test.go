package valkey

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	libvalkey "github.com/valkey-io/valkey-go"
	"github.com/valkey-io/valkey-go/mock"
	"go.uber.org/mock/gomock"
)

func getTestSrvOptions() SrvOptions {
	return SrvOptions{
		InitAddress: []string{"test.valkey.invalid:6379"},
		Username:    "test_user",
		Password:    "test_password",
		SelectDB:    0,
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	srvOpts := getTestSrvOptions()

	got, err := New(
		t.Context(),
		srvOpts,
		WithMessageEncodeFunc(nil),
	)

	require.Error(t, err)
	require.Nil(t, got)

	got, err = New(
		t.Context(),
		srvOpts,
	)

	require.Error(t, err)
	require.Nil(t, got)

	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })

	vkc := mock.NewClient(ctrl)

	got, err = New(
		t.Context(),
		srvOpts,
		WithValkeyClient(vkc),
	)

	require.NoError(t, err)
	require.NotNil(t, got)

	// Close is idempotent: the underlying client is released at most once even
	// when Close is called repeatedly (mock EXPECT defaults to exactly one call).
	vkc.EXPECT().Close()

	got.Close()
	got.Close()
}

func TestSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     string
		val     string
		exp     time.Duration
		mock    func(ctx context.Context, vkc *mock.Client)
		wantErr bool
	}{
		{
			name: "success",
			key:  "key1",
			val:  "val1",
			exp:  time.Second,
			mock: func(ctx context.Context, vkc *mock.Client) {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("SET", "key1", "val1", "EX", "1"),
				)
			},
			wantErr: false,
		},
		{
			name: "error",
			key:  "key2",
			val:  "val2",
			exp:  2 * time.Second,
			mock: func(ctx context.Context, vkc *mock.Client) {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("SET", "key2", "val2", "EX", "2"),
				).Return(mock.ErrorResult(errors.New("error")))
			},
			wantErr: true,
		},
		{
			name: "success without expiration",
			key:  "key3",
			val:  "val3",
			exp:  0,
			mock: func(ctx context.Context, vkc *mock.Client) {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("SET", "key3", "val3"),
				)
			},
			wantErr: false,
		},
		{
			name: "success with sub-second expiration",
			key:  "key4",
			val:  "val4",
			exp:  500 * time.Millisecond,
			mock: func(ctx context.Context, vkc *mock.Client) {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("SET", "key4", "val4", "PX", "500"),
				)
			},
			wantErr: false,
		},
		{
			name: "success with non-whole-second expiration",
			key:  "key5",
			val:  "val5",
			exp:  1500 * time.Millisecond,
			mock: func(ctx context.Context, vkc *mock.Client) {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("SET", "key5", "val5", "PX", "1500"),
				)
			},
			wantErr: false,
		},
		{
			name: "success with sub-millisecond expiration clamped to 1ms",
			key:  "key6",
			val:  "val6",
			exp:  100 * time.Microsecond,
			mock: func(ctx context.Context, vkc *mock.Client) {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("SET", "key6", "val6", "PX", "1"),
				)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srvOpts := getTestSrvOptions()

			ctrl := gomock.NewController(t)
			t.Cleanup(func() { ctrl.Finish() })

			vkc := mock.NewClient(ctrl)
			ctx := t.Context()

			cli, err := New(
				ctx,
				srvOpts,
				WithValkeyClient(vkc),
			)

			require.NoError(t, err)
			require.NotNil(t, cli)

			tt.mock(ctx, vkc)

			err = cli.Set(ctx, tt.key, tt.val, tt.exp)
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
		name      string
		key       string
		val       string
		mock      func(ctx context.Context, vkc *mock.Client)
		wantErr   bool
		wantErrIs error
	}{
		{
			name: "success",
			key:  "key1",
			val:  "val1",
			mock: func(ctx context.Context, vkc *mock.Client) {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("GET", "key1"),
				).Return(mock.Result(mock.ValkeyString("val1")))
			},
			wantErr: false,
		},
		{
			name: "error",
			key:  "key2",
			val:  "val2",
			mock: func(ctx context.Context, vkc *mock.Client) {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("GET", "key2"),
				).Return(mock.ErrorResult(errors.New("error")))
			},
			wantErr: true,
		},
		{
			name: "key not found",
			key:  "key3",
			mock: func(ctx context.Context, vkc *mock.Client) {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("GET", "key3"),
				).Return(mock.Result(mock.ValkeyNil()))
			},
			wantErr:   true,
			wantErrIs: ErrKeyNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srvOpts := getTestSrvOptions()

			ctrl := gomock.NewController(t)
			t.Cleanup(func() { ctrl.Finish() })

			vkc := mock.NewClient(ctrl)
			ctx := t.Context()

			cli, err := New(
				ctx,
				srvOpts,
				WithValkeyClient(vkc),
			)

			require.NoError(t, err)
			require.NotNil(t, cli)

			tt.mock(ctx, vkc)

			val, err := cli.Get(ctx, tt.key)
			if tt.wantErr {
				require.Error(t, err)

				if tt.wantErrIs != nil {
					require.ErrorIs(t, err, tt.wantErrIs)
				}

				require.Empty(t, val)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.val, val)
		})
	}
}

func TestDel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     string
		mock    func(ctx context.Context, vkc *mock.Client)
		wantErr bool
	}{
		{
			name: "success",
			key:  "key1",
			mock: func(ctx context.Context, vkc *mock.Client) {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("DEL", "key1"),
				)
			},
			wantErr: false,
		},
		{
			name: "error",
			key:  "key2",
			mock: func(ctx context.Context, vkc *mock.Client) {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("DEL", "key2"),
				).Return(mock.ErrorResult(errors.New("error")))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srvOpts := getTestSrvOptions()

			ctrl := gomock.NewController(t)
			t.Cleanup(func() { ctrl.Finish() })

			vkc := mock.NewClient(ctrl)
			ctx := t.Context()

			cli, err := New(
				ctx,
				srvOpts,
				WithValkeyClient(vkc),
			)

			require.NoError(t, err)
			require.NotNil(t, cli)

			tt.mock(ctx, vkc)

			err = cli.Del(ctx, tt.key)
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
		name    string
		channel string
		message string
		mock    func(ctx context.Context, vkc *mock.Client)
		wantErr bool
	}{
		{
			name:    "success",
			channel: "ch1",
			message: "msg1",
			mock: func(ctx context.Context, vkc *mock.Client) {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("PUBLISH", "ch1", "msg1"),
				)
			},
			wantErr: false,
		},
		{
			name:    "error",
			channel: "ch2",
			message: "msg2",
			mock: func(ctx context.Context, vkc *mock.Client) {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("PUBLISH", "ch2", "msg2"),
				).Return(mock.ErrorResult(errors.New("error")))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srvOpts := getTestSrvOptions()

			ctrl := gomock.NewController(t)
			t.Cleanup(func() { ctrl.Finish() })

			vkc := mock.NewClient(ctrl)
			ctx := t.Context()

			cli, err := New(
				ctx,
				srvOpts,
				WithValkeyClient(vkc),
			)

			require.NoError(t, err)
			require.NotNil(t, cli)

			tt.mock(ctx, vkc)

			err = cli.Send(ctx, tt.channel, tt.message)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestReceive(t *testing.T) {
	t.Parallel()

	t.Run("no channels configured", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(func() { ctrl.Finish() })

		vkc := mock.NewClient(ctrl)
		ctx := t.Context()

		cli, err := New(
			ctx,
			getTestSrvOptions(),
			WithValkeyClient(vkc),
		)

		require.NoError(t, err)
		require.NotNil(t, cli)

		channel, message, err := cli.Receive(ctx)
		require.ErrorIs(t, err, ErrNoSubscription)
		require.Empty(t, channel)
		require.Empty(t, message)
	})

	t.Run("one message per call", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(func() { ctrl.Finish() })

		vkc := mock.NewClient(ctrl)
		ctx := t.Context()

		// The blocking valkey-go Receive runs once in the background
		// subscription goroutine and invokes the callback for every
		// published message.
		vkc.EXPECT().Receive(
			gomock.Any(),
			mock.Match("SUBSCRIBE", "ch1", "ch2"),
			gomock.Any(),
		).Do(func(_, _ any, fn func(message VKMessage)) {
			fn(VKMessage{Channel: "ch1", Message: "msg1"})
			fn(VKMessage{Channel: "ch2", Message: "msg2"})
		})

		cli, err := New(
			ctx,
			getTestSrvOptions(),
			WithValkeyClient(vkc),
			WithChannels("ch1", "ch2"),
		)

		require.NoError(t, err)
		require.NotNil(t, cli)

		channel, message, err := cli.Receive(ctx)
		require.NoError(t, err)
		require.Equal(t, "ch1", channel)
		require.Equal(t, "msg1", message)

		channel, message, err = cli.Receive(ctx)
		require.NoError(t, err)
		require.Equal(t, "ch2", channel)
		require.Equal(t, "msg2", message)

		// The subscription terminated cleanly after the last message.
		channel, message, err = cli.Receive(ctx)
		require.ErrorIs(t, err, ErrSubscriptionClosed)
		require.Empty(t, channel)
		require.Empty(t, message)
	})

	t.Run("subscription error", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(func() { ctrl.Finish() })

		vkc := mock.NewClient(ctrl)
		ctx := t.Context()

		vkc.EXPECT().Receive(
			gomock.Any(),
			mock.Match("SUBSCRIBE", "ch1", "ch2"),
			gomock.Any(),
		).Return(errors.New("mock receive error"))

		cli, err := New(
			ctx,
			getTestSrvOptions(),
			WithValkeyClient(vkc),
			WithChannels("ch1", "ch2"),
		)

		require.NoError(t, err)
		require.NotNil(t, cli)

		channel, message, err := cli.Receive(ctx)
		require.Error(t, err)
		require.ErrorContains(t, err, "mock receive error")
		require.Empty(t, channel)
		require.Empty(t, message)
	})

	t.Run("context canceled", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(func() { ctrl.Finish() })

		vkc := mock.NewClient(ctrl)
		ctx := t.Context()

		started := make(chan struct{})
		release := make(chan struct{})

		t.Cleanup(func() { close(release) })

		// Simulate a live subscription with no incoming messages.
		vkc.EXPECT().Receive(
			gomock.Any(),
			mock.Match("SUBSCRIBE", "ch1", "ch2"),
			gomock.Any(),
		).Do(func(_, _ any, _ func(message VKMessage)) {
			close(started)
			<-release
		})

		cli, err := New(
			ctx,
			getTestSrvOptions(),
			WithValkeyClient(vkc),
			WithChannels("ch1", "ch2"),
		)

		require.NoError(t, err)
		require.NotNil(t, cli)

		// Wait for the background subscription goroutine to be running.
		<-started

		cctx, cancel := context.WithCancel(ctx)
		cancel()

		channel, message, err := cli.Receive(cctx)
		require.ErrorIs(t, err, context.Canceled)
		require.Empty(t, channel)
		require.Empty(t, message)
	})
}

// TestReceiveClose covers Receive behavior around a deliberate Close.
func TestReceiveClose(t *testing.T) {
	t.Parallel()

	t.Run("close terminates the subscription", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(func() { ctrl.Finish() })

		vkc := mock.NewClient(ctrl)
		ctx := t.Context()

		// Block like the real client until the subscription context is
		// canceled by Close, then return its error.
		vkc.EXPECT().Receive(
			gomock.Any(),
			mock.Match("SUBSCRIBE", "ch1"),
			gomock.Any(),
		).DoAndReturn(func(rctx context.Context, _, _ any) error {
			<-rctx.Done()

			return rctx.Err()
		})

		vkc.EXPECT().Close()

		cli, err := New(
			ctx,
			getTestSrvOptions(),
			WithValkeyClient(vkc),
			WithChannels("ch1"),
		)

		require.NoError(t, err)
		require.NotNil(t, cli)

		cli.Close()

		// A Close-driven context cancellation is reported as a clean closure,
		// not wrapped as a receive error.
		channel, message, err := cli.Receive(ctx)
		require.ErrorIs(t, err, ErrSubscriptionClosed)
		require.Empty(t, channel)
		require.Empty(t, message)
	})

	t.Run("close reports clean closure even when the client returns ErrClosing", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(func() { ctrl.Finish() })

		vkc := mock.NewClient(ctrl)
		ctx := t.Context()

		// On Close the live client tears down via both the context cancellation
		// and vkclient.Close(); the pipe races between them and can return
		// ErrClosing instead of ctx.Err(). Receive must still report a clean
		// closure rather than wrapping ErrClosing as a receive error.
		vkc.EXPECT().Receive(
			gomock.Any(),
			mock.Match("SUBSCRIBE", "ch1"),
			gomock.Any(),
		).DoAndReturn(func(rctx context.Context, _, _ any) error {
			<-rctx.Done()

			return libvalkey.ErrClosing
		})

		vkc.EXPECT().Close()

		cli, err := New(
			ctx,
			getTestSrvOptions(),
			WithValkeyClient(vkc),
			WithChannels("ch1"),
		)

		require.NoError(t, err)
		require.NotNil(t, cli)

		cli.Close()

		channel, message, err := cli.Receive(ctx)
		require.ErrorIs(t, err, ErrSubscriptionClosed)
		require.Empty(t, channel)
		require.Empty(t, message)
	})

	t.Run("close drops a message blocked on delivery", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(func() { ctrl.Finish() })

		vkc := mock.NewClient(ctrl)
		ctx := t.Context()

		started := make(chan struct{})
		delivered := make(chan struct{})

		// Publish a message that no Receive call ever consumes: the delivery
		// callback stays blocked on the internal channel until Close cancels
		// the subscription context, which must unblock it and drop the message.
		vkc.EXPECT().Receive(
			gomock.Any(),
			mock.Match("SUBSCRIBE", "ch1"),
			gomock.Any(),
		).DoAndReturn(func(rctx context.Context, _ any, fn func(message VKMessage)) error {
			close(started)
			fn(VKMessage{Channel: "ch1", Message: "msg1"})
			close(delivered)

			return rctx.Err()
		})

		vkc.EXPECT().Close()

		cli, err := New(
			ctx,
			getTestSrvOptions(),
			WithValkeyClient(vkc),
			WithChannels("ch1"),
		)

		require.NoError(t, err)
		require.NotNil(t, cli)

		// Wait for the background subscription goroutine to be running, then
		// tear it down while the undelivered message is still pending.
		<-started
		cli.Close()

		// Close must unblock the delivery callback without a consumer.
		<-delivered

		channel, message, err := cli.Receive(ctx)
		require.ErrorIs(t, err, ErrSubscriptionClosed)
		require.Empty(t, channel)
		require.Empty(t, message)
	})

	t.Run("close during a concurrent receive", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		t.Cleanup(func() { ctrl.Finish() })

		vkc := mock.NewClient(ctrl)
		ctx := t.Context()

		started := make(chan struct{})

		// Simulate a live, idle subscription that terminates only when Close
		// cancels the subscription context.
		vkc.EXPECT().Receive(
			gomock.Any(),
			mock.Match("SUBSCRIBE", "ch1"),
			gomock.Any(),
		).DoAndReturn(func(rctx context.Context, _, _ any) error {
			close(started)
			<-rctx.Done()

			return rctx.Err()
		})

		vkc.EXPECT().Close()

		cli, err := New(
			ctx,
			getTestSrvOptions(),
			WithValkeyClient(vkc),
			WithChannels("ch1"),
		)

		require.NoError(t, err)
		require.NotNil(t, cli)

		// Wait for the background subscription goroutine to be running, then
		// block a Receive on it while Close tears the subscription down. Both
		// interleavings (Receive blocked first or Close completing first) must
		// end in a clean closure; the race detector checks the flag handoff.
		<-started

		type received struct {
			channel string
			message string
			err     error
		}

		resch := make(chan received, 1)

		go func() {
			channel, message, rerr := cli.Receive(ctx)
			resch <- received{channel: channel, message: message, err: rerr}
		}()

		cli.Close()

		res := <-resch
		require.ErrorIs(t, res.err, ErrSubscriptionClosed)
		require.Empty(t, res.channel)
		require.Empty(t, res.message)
	})
}

func TestSetData(t *testing.T) {
	t.Parallel()

	type TestData struct {
		Alpha string
		Beta  int
	}

	testMsg := TestData{Alpha: "abc123", Beta: -567}
	testEncMsg, err := MessageEncode(testMsg)

	require.NoError(t, err)

	tests := []struct {
		name    string
		key     string
		val     any
		exp     time.Duration
		mock    func(ctx context.Context, vkc *mock.Client)
		wantErr bool
	}{
		{
			name: "success",
			key:  "key1",
			val:  testMsg,
			exp:  2 * time.Second,
			mock: func(ctx context.Context, vkc *mock.Client) {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("SET", "key1", testEncMsg, "EX", "2"),
				)
			},
			wantErr: false,
		},
		{
			name: "error",
			key:  "key2",
			val:  testMsg,
			exp:  time.Second,
			mock: func(ctx context.Context, vkc *mock.Client) {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("SET", "key2", testEncMsg, "EX", "1"),
				).Return(mock.ErrorResult(errors.New("error")))
			},
			wantErr: true,
		},
		{
			name:    "data error",
			key:     "key2",
			val:     nil,
			exp:     time.Second,
			mock:    func(_ context.Context, _ *mock.Client) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srvOpts := getTestSrvOptions()

			ctrl := gomock.NewController(t)
			t.Cleanup(func() { ctrl.Finish() })

			vkc := mock.NewClient(ctrl)
			ctx := t.Context()

			cli, err := New(
				ctx,
				srvOpts,
				WithValkeyClient(vkc),
			)

			require.NoError(t, err)
			require.NotNil(t, cli)

			tt.mock(ctx, vkc)

			err = cli.SetData(ctx, tt.key, tt.val, tt.exp)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGetData(t *testing.T) {
	t.Parallel()

	type TestData struct {
		Alpha string
		Beta  int
	}

	testMsg := TestData{Alpha: "abc123", Beta: -567}
	testEncMsg, err := MessageEncode(testMsg)

	require.NoError(t, err)

	tests := []struct {
		name      string
		key       string
		val       any
		mock      func(ctx context.Context, vkc *mock.Client)
		wantErr   bool
		wantErrIs error
	}{
		{
			name: "success",
			key:  "key1",
			val:  testMsg,
			mock: func(ctx context.Context, vkc *mock.Client) {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("GET", "key1"),
				).Return(mock.Result(mock.ValkeyString(testEncMsg)))
			},
			wantErr: false,
		},
		{
			name: "error",
			key:  "key2",
			val:  TestData{},
			mock: func(ctx context.Context, vkc *mock.Client) {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("GET", "key2"),
				).Return(mock.ErrorResult(errors.New("error")))
			},
			wantErr: true,
		},
		{
			name: "data error",
			key:  "key3",
			val:  TestData{},
			mock: func(ctx context.Context, vkc *mock.Client) {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("GET", "key3"),
				).Return(mock.Result(mock.ValkeyString("INVALID-CORRUPT-DATA")))
			},
			wantErr: true,
		},
		{
			name: "key not found",
			key:  "key4",
			val:  TestData{},
			mock: func(ctx context.Context, vkc *mock.Client) {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("GET", "key4"),
				).Return(mock.Result(mock.ValkeyNil()))
			},
			wantErr:   true,
			wantErrIs: ErrKeyNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srvOpts := getTestSrvOptions()

			ctrl := gomock.NewController(t)
			t.Cleanup(func() { ctrl.Finish() })

			vkc := mock.NewClient(ctrl)
			ctx := t.Context()

			cli, err := New(
				ctx,
				srvOpts,
				WithValkeyClient(vkc),
			)

			require.NoError(t, err)
			require.NotNil(t, cli)

			tt.mock(ctx, vkc)

			var data TestData

			err = cli.GetData(ctx, tt.key, &data)
			if tt.wantErr {
				require.Error(t, err)

				if tt.wantErrIs != nil {
					require.ErrorIs(t, err, tt.wantErrIs)
				}

				require.Empty(t, data)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.val, data)
		})
	}
}

func TestSendData(t *testing.T) {
	t.Parallel()

	type TestData struct {
		Alpha string
		Beta  int
	}

	testMsg := TestData{Alpha: "abc123", Beta: -567}
	testEncMsg, err := MessageEncode(testMsg)

	require.NoError(t, err)

	tests := []struct {
		name    string
		channel string
		message any
		mock    func(ctx context.Context, vkc *mock.Client)
		wantErr bool
	}{
		{
			name:    "success",
			channel: "ch1",
			message: testMsg,
			mock: func(ctx context.Context, vkc *mock.Client) {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("PUBLISH", "ch1", testEncMsg),
				)
			},
			wantErr: false,
		},
		{
			name:    "error",
			channel: "ch2",
			message: testMsg,
			mock: func(ctx context.Context, vkc *mock.Client) {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("PUBLISH", "ch2", testEncMsg),
				).Return(mock.ErrorResult(errors.New("error")))
			},
			wantErr: true,
		},
		{
			name:    "data error",
			channel: "ch2",
			message: nil,
			mock:    func(_ context.Context, _ *mock.Client) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srvOpts := getTestSrvOptions()

			ctrl := gomock.NewController(t)
			t.Cleanup(func() { ctrl.Finish() })

			vkc := mock.NewClient(ctrl)
			ctx := t.Context()

			cli, err := New(
				ctx,
				srvOpts,
				WithValkeyClient(vkc),
			)

			require.NoError(t, err)
			require.NotNil(t, cli)

			tt.mock(ctx, vkc)

			err = cli.SendData(ctx, tt.channel, tt.message)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestReceiveData(t *testing.T) {
	t.Parallel()

	type TestData struct {
		Alpha string
		Beta  int
	}

	testMsg := TestData{Alpha: "abc123", Beta: -567}
	testEncMsg, err := MessageEncode(testMsg)

	require.NoError(t, err)

	tests := []struct {
		name    string
		channel string
		message any
		mock    func(ctx context.Context, vkc *mock.Client)
		wantErr bool
	}{
		{
			name:    "success",
			channel: "ch1",
			message: testMsg,
			mock: func(_ context.Context, vkc *mock.Client) {
				vkc.EXPECT().Receive(
					gomock.Any(),
					mock.Match("SUBSCRIBE", "ch1", "ch2"),
					gomock.Any(),
				).Do(func(_, _ any, fn func(message VKMessage)) {
					fn(VKMessage{Channel: "ch1", Message: testEncMsg})
				})
			},
			wantErr: false,
		},
		{
			name:    "error",
			channel: "ch2",
			message: testMsg,
			mock: func(_ context.Context, vkc *mock.Client) {
				vkc.EXPECT().Receive(
					gomock.Any(),
					mock.Match("SUBSCRIBE", "ch1", "ch2"),
					gomock.Any(),
				).Return(errors.New("error"))
			},
			wantErr: true,
		},
		{
			name:    "data error",
			channel: "ch2",
			message: TestData{},
			mock: func(_ context.Context, vkc *mock.Client) {
				vkc.EXPECT().Receive(
					gomock.Any(),
					mock.Match("SUBSCRIBE", "ch1", "ch2"),
					gomock.Any(),
				).Do(func(_, _ any, fn func(message VKMessage)) {
					fn(VKMessage{Channel: "ch3", Message: "INVALID-CORRUPT-DATA"})
				})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srvOpts := getTestSrvOptions()

			ctrl := gomock.NewController(t)
			t.Cleanup(func() { ctrl.Finish() })

			vkc := mock.NewClient(ctrl)
			ctx := t.Context()

			// The background subscription starts inside New, so the mock
			// expectation must be registered before creating the client.
			tt.mock(ctx, vkc)

			cli, err := New(
				ctx,
				srvOpts,
				WithValkeyClient(vkc),
				WithChannels("ch1", "ch2"),
			)

			require.NoError(t, err)
			require.NotNil(t, cli)

			var data TestData

			channel, err := cli.ReceiveData(ctx, &data)
			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, data)
				require.Empty(t, channel)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.channel, channel)
			require.Equal(t, tt.message, data)
		})
	}
}

func TestHealthCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mock    func(ctx context.Context, vkc *mock.Client)
		wantErr bool
	}{
		{
			name: "success",
			mock: func(ctx context.Context, vkc *mock.Client) {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("PING"),
				)
			},
			wantErr: false,
		},
		{
			name: "error",
			mock: func(ctx context.Context, vkc *mock.Client) {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("PING"),
				).Return(mock.ErrorResult(errors.New("error")))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srvOpts := getTestSrvOptions()

			ctrl := gomock.NewController(t)
			t.Cleanup(func() { ctrl.Finish() })

			vkc := mock.NewClient(ctrl)
			ctx := t.Context()

			cli, err := New(
				ctx,
				srvOpts,
				WithValkeyClient(vkc),
			)

			require.NoError(t, err)
			require.NotNil(t, cli)

			tt.mock(ctx, vkc)

			err = cli.HealthCheck(ctx)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
