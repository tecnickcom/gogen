package redis

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_loadConfig(t *testing.T) {
	t.Parallel()

	srvOpts := &SrvOptions{
		Addr:     "test.redis.invalid:6379",
		Username: "test_user",
		Password: "test_password",
		DB:       0,
	}

	got, err := loadConfig(
		context.TODO(),
		srvOpts,
		WithMessageEncodeFunc(DefaultMessageEncodeFunc),
		WithMessageDecodeFunc(DefaultMessageDecodeFunc),
		WithSubscrChannels("test_channel_1", "test_channel_2"),
	)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, srvOpts.Addr, got.srvOpts.Addr)
	require.Equal(t, srvOpts.Username, got.srvOpts.Username)
	require.Equal(t, srvOpts.Password, got.srvOpts.Password)
	require.Equal(t, srvOpts.DB, got.srvOpts.DB)
	require.NotNil(t, got.messageEncodeFunc)
	require.NotNil(t, got.messageDecodeFunc)

	got, err = loadConfig(
		context.TODO(),
		nil,
	)

	require.Error(t, err)
	require.Nil(t, got)

	got, err = loadConfig(
		context.TODO(),
		&SrvOptions{},
	)

	require.Error(t, err)
	require.Nil(t, got)

	got, err = loadConfig(
		context.TODO(),
		srvOpts,
		WithMessageEncodeFunc(nil),
	)

	require.Error(t, err)
	require.Nil(t, got)

	got, err = loadConfig(
		context.TODO(),
		srvOpts,
		WithMessageDecodeFunc(nil),
	)

	require.Error(t, err)
	require.Nil(t, got)
}
