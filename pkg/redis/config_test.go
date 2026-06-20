package redis

import (
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
		nil,
	)

	require.Error(t, err)
	require.Nil(t, got)

	got, err = loadConfig(
		&SrvOptions{},
	)

	require.Error(t, err)
	require.Nil(t, got)

	got, err = loadConfig(
		srvOpts,
		WithMessageEncodeFunc(nil),
	)

	require.Error(t, err)
	require.Nil(t, got)

	got, err = loadConfig(
		srvOpts,
		WithMessageDecodeFunc(nil),
	)

	require.Error(t, err)
	require.Nil(t, got)
}

func Test_loadConfig_addrValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{
			name:    "valid host and port",
			addr:    "localhost:6379",
			wantErr: false,
		},
		{
			name:    "valid IPv6 host and port",
			addr:    "[::1]:6379",
			wantErr: false,
		},
		{
			name:    "valid single-digit port",
			addr:    "localhost:1",
			wantErr: false,
		},
		{
			name:    "valid omitted host",
			addr:    ":6379",
			wantErr: false,
		},
		{
			name:    "invalid missing port",
			addr:    "localhost",
			wantErr: true,
		},
		{
			name:    "invalid empty address",
			addr:    "",
			wantErr: true,
		},
		{
			name:    "invalid missing port value",
			addr:    "localhost:",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := loadConfig(&SrvOptions{Addr: tt.addr})
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
		})
	}
}
