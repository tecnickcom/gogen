package valkey

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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

	require.Error(t, err)
	require.Nil(t, got)

	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })

	vkc := mock.NewClient(ctrl)

	got, err = New(
		context.TODO(),
		srvOpts,
		WithValkeyClient(vkc),
	)

	require.NoError(t, err)
	require.NotNil(t, got)

	vkc.EXPECT().Close()

	got.Close()
}

func TestSet(t *testing.T) {
	t.Parallel()

	srvOpts := getTestSrvOptions()

	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })

	vkc := mock.NewClient(ctrl)
	ctx := context.TODO()

	cli, err := New(
		ctx,
		srvOpts,
		WithValkeyClient(vkc),
	)

	require.NoError(t, err)
	require.NotNil(t, cli)

	tests := []struct {
		name    string
		key     string
		val     string
		exp     time.Duration
		mock    func()
		wantErr bool
	}{
		{
			name: "success",
			key:  "key1",
			val:  "val1",
			exp:  time.Second,
			mock: func() {
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
			mock: func() {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("SET", "key2", "val2", "EX", "2"),
				).Return(mock.ErrorResult(errors.New("error")))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.mock()

			err := cli.Set(ctx, tt.key, tt.val, tt.exp)
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

	srvOpts := getTestSrvOptions()

	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })

	vkc := mock.NewClient(ctrl)
	ctx := context.TODO()

	cli, err := New(
		ctx,
		srvOpts,
		WithValkeyClient(vkc),
	)

	require.NoError(t, err)
	require.NotNil(t, cli)

	tests := []struct {
		name    string
		key     string
		val     string
		mock    func()
		wantErr bool
	}{
		{
			name: "success",
			key:  "key1",
			val:  "val1",
			mock: func() {
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
			mock: func() {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("GET", "key2"),
				).Return(mock.ErrorResult(errors.New("error")))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.mock()

			val, err := cli.Get(ctx, tt.key)
			if tt.wantErr {
				require.Error(t, err)
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

	srvOpts := getTestSrvOptions()

	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })

	vkc := mock.NewClient(ctrl)
	ctx := context.TODO()

	cli, err := New(
		ctx,
		srvOpts,
		WithValkeyClient(vkc),
	)

	require.NoError(t, err)
	require.NotNil(t, cli)

	tests := []struct {
		name    string
		key     string
		mock    func()
		wantErr bool
	}{
		{
			name: "success",
			key:  "key1",
			mock: func() {
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
			mock: func() {
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

			tt.mock()

			err := cli.Del(ctx, tt.key)
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

	srvOpts := getTestSrvOptions()

	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })

	vkc := mock.NewClient(ctrl)
	ctx := context.TODO()

	cli, err := New(
		ctx,
		srvOpts,
		WithValkeyClient(vkc),
	)

	require.NoError(t, err)
	require.NotNil(t, cli)

	tests := []struct {
		name    string
		channel string
		message string
		mock    func()
		wantErr bool
	}{
		{
			name:    "success",
			channel: "ch1",
			message: "msg1",
			mock: func() {
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
			mock: func() {
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

			tt.mock()

			err := cli.Send(ctx, tt.channel, tt.message)
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

	srvOpts := getTestSrvOptions()

	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })

	vkc := mock.NewClient(ctrl)
	ctx := context.TODO()

	cli, err := New(
		ctx,
		srvOpts,
		WithValkeyClient(vkc),
		WithChannels("ch1", "ch2"),
	)

	require.NoError(t, err)
	require.NotNil(t, cli)

	tests := []struct {
		name    string
		channel string
		message string
		mock    func()
		wantErr bool
	}{
		{
			name:    "success",
			channel: "ch1",
			message: "msg1",
			mock: func() {
				vkc.EXPECT().Receive(
					ctx,
					mock.Match("SUBSCRIBE", "ch1", "ch2"),
					gomock.Any(),
				).Do(func(_, _ any, fn func(message VKMessage)) {
					fn(VKMessage{Channel: "ch1", Message: "msg1"})
				})
			},
			wantErr: false,
		},
		{
			name:    "error",
			channel: "ch2",
			message: "msg2",
			mock: func() {
				vkc.EXPECT().Receive(
					ctx,
					mock.Match("SUBSCRIBE", "ch1", "ch2"),
					gomock.Any(),
				).Do(func(_, _ any, fn func(message VKMessage)) {
					fn(VKMessage{Channel: "ch2", Message: "msg2"})
				}).Return(errors.New("error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.mock()

			channel, message, err := cli.Receive(ctx)
			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, channel)
				require.Empty(t, message)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.channel, channel)
			require.Equal(t, tt.message, message)
		})
	}
}

func TestSetData(t *testing.T) {
	t.Parallel()

	srvOpts := getTestSrvOptions()

	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })

	vkc := mock.NewClient(ctrl)
	ctx := context.TODO()

	cli, err := New(
		ctx,
		srvOpts,
		WithValkeyClient(vkc),
	)

	require.NoError(t, err)
	require.NotNil(t, cli)

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
		mock    func()
		wantErr bool
	}{
		{
			name: "success",
			key:  "key1",
			val:  testMsg,
			exp:  2 * time.Second,
			mock: func() {
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
			mock: func() {
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
			mock:    func() {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.mock()

			err := cli.SetData(ctx, tt.key, tt.val, tt.exp)
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

	srvOpts := getTestSrvOptions()

	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })

	vkc := mock.NewClient(ctrl)
	ctx := context.TODO()

	cli, err := New(
		ctx,
		srvOpts,
		WithValkeyClient(vkc),
	)

	require.NoError(t, err)
	require.NotNil(t, cli)

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
		mock    func()
		wantErr bool
	}{
		{
			name: "success",
			key:  "key1",
			val:  testMsg,
			mock: func() {
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
			mock: func() {
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
			mock: func() {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("GET", "key3"),
				).Return(mock.Result(mock.ValkeyString("INVALID-CORRUPT-DATA")))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.mock()

			var data TestData

			err := cli.GetData(ctx, tt.key, &data)
			if tt.wantErr {
				require.Error(t, err)
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

	srvOpts := getTestSrvOptions()

	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })

	vkc := mock.NewClient(ctrl)
	ctx := context.TODO()

	cli, err := New(
		ctx,
		srvOpts,
		WithValkeyClient(vkc),
	)

	require.NoError(t, err)
	require.NotNil(t, cli)

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
		mock    func()
		wantErr bool
	}{
		{
			name:    "success",
			channel: "ch1",
			message: testMsg,
			mock: func() {
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
			mock: func() {
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
			mock:    func() {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.mock()

			err := cli.SendData(ctx, tt.channel, tt.message)
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

	srvOpts := getTestSrvOptions()

	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })

	vkc := mock.NewClient(ctrl)
	ctx := context.TODO()

	cli, err := New(
		ctx,
		srvOpts,
		WithValkeyClient(vkc),
		WithChannels("ch1", "ch2"),
	)

	require.NoError(t, err)
	require.NotNil(t, cli)

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
		mock    func()
		wantErr bool
	}{
		{
			name:    "success",
			channel: "ch1",
			message: testMsg,
			mock: func() {
				vkc.EXPECT().Receive(
					ctx,
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
			mock: func() {
				vkc.EXPECT().Receive(
					ctx,
					mock.Match("SUBSCRIBE", "ch1", "ch2"),
					gomock.Any(),
				).Do(func(_, _ any, fn func(message VKMessage)) {
					fn(VKMessage{Channel: "ch2", Message: testEncMsg})
				}).Return(errors.New("error"))
			},
			wantErr: true,
		},
		{
			name:    "data error",
			channel: "ch2",
			message: TestData{},
			mock: func() {
				vkc.EXPECT().Receive(
					ctx,
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

			tt.mock()

			var data TestData

			channel, err := cli.ReceiveData(ctx, &data)
			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, data)

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

	srvOpts := getTestSrvOptions()

	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })

	vkc := mock.NewClient(ctrl)
	ctx := context.TODO()

	cli, err := New(
		ctx,
		srvOpts,
		WithValkeyClient(vkc),
	)

	require.NoError(t, err)
	require.NotNil(t, cli)

	tests := []struct {
		name    string
		mock    func()
		wantErr bool
	}{
		{
			name: "success",
			mock: func() {
				vkc.EXPECT().Do(
					ctx,
					mock.Match("PING"),
				)
			},
			wantErr: false,
		},
		{
			name: "error",
			mock: func() {
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

			tt.mock()

			err := cli.HealthCheck(ctx)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
