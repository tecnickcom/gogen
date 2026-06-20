package valkey

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_loadConfig(t *testing.T) {
	t.Parallel()

	srvOpts := SrvOptions{
		InitAddress: []string{"test.valkey.invalid:6379"},
		Username:    "test_user",
		Password:    "test_password",
		SelectDB:    0,
	}

	got, err := loadConfig(
		t.Context(),
		srvOpts,
		WithMessageEncodeFunc(DefaultMessageEncodeFunc),
		WithMessageDecodeFunc(DefaultMessageDecodeFunc),
		WithChannels("test_channel_1", "test_channel_2"),
	)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, srvOpts.InitAddress, got.srvOpts.InitAddress)
	require.Equal(t, srvOpts.Username, got.srvOpts.Username)
	require.Equal(t, srvOpts.Password, got.srvOpts.Password)
	require.Equal(t, srvOpts.SelectDB, got.srvOpts.SelectDB)
	require.NotNil(t, got.messageEncodeFunc)
	require.NotNil(t, got.messageDecodeFunc)

	got, err = loadConfig(
		t.Context(),
		SrvOptions{},
	)

	require.Error(t, err)
	require.Nil(t, got)

	got, err = loadConfig(
		t.Context(),
		srvOpts,
		WithMessageEncodeFunc(nil),
	)

	require.Error(t, err)
	require.Nil(t, got)

	got, err = loadConfig(
		t.Context(),
		srvOpts,
		WithMessageDecodeFunc(nil),
	)

	require.Error(t, err)
	require.Nil(t, got)
}

func Test_validInitAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		addrs []string
		want  bool
	}{
		{
			name:  "nil address list",
			addrs: nil,
			want:  false,
		},
		{
			name:  "empty address list",
			addrs: []string{},
			want:  false,
		},
		{
			name:  "valid hostname and port",
			addrs: []string{"test.valkey.invalid:6379"},
			want:  true,
		},
		{
			name:  "valid single-digit port",
			addrs: []string{"localhost:6"},
			want:  true,
		},
		{
			name:  "valid IPv6 address",
			addrs: []string{"[::1]:6379"},
			want:  true,
		},
		{
			name:  "valid multiple addresses",
			addrs: []string{"localhost:6379", "[::1]:6380"},
			want:  true,
		},
		{
			name:  "missing port",
			addrs: []string{"localhost"},
			want:  false,
		},
		{
			name:  "missing host",
			addrs: []string{":6379"},
			want:  false,
		},
		{
			name:  "empty port",
			addrs: []string{"localhost:"},
			want:  false,
		},
		{
			name:  "malformed address",
			addrs: []string{"::::"},
			want:  false,
		},
		{
			name:  "one invalid address in list",
			addrs: []string{"localhost:6379", "localhost"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, validInitAddress(tt.addrs))
		})
	}
}
