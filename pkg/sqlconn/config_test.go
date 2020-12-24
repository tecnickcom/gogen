package sqlconn

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_config_validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config
		wantErr bool
	}{
		{
			name:    "fail with empty driver",
			cfg:     defaultConfig("", "user:pass@tcp(127.0.0.1:1234)/testdb"),
			wantErr: true,
		},
		{
			name:    "fail with empty DSN",
			cfg:     defaultConfig("sqldb", ""),
			wantErr: true,
		},
		{
			name: "fail with invalid max retry value",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.connectMaxRetry = 0
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "fail with invalid retry interval",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.connectRetryInterval = 0
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "fail with invalid connect function",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.connectFunc = nil
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "fail with invalid check connection function",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.checkConnectionFunc = nil
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "fail with invalid sql open function",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.sqlOpenFunc = nil
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "fail with invalid max idle",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.connMaxIdle = 0
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "succeed with no errors",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				return cfg
			}(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := tt.cfg.validate(); (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_defaultConfig(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig("test_driver", "test_dsn")
	require.NotNil(t, cfg)
	require.Equal(t, "test_driver", cfg.driver)
	require.Equal(t, "test_dsn", cfg.dsn)
	require.NotNil(t, cfg.connectFunc)
	require.NotNil(t, cfg.checkConnectionFunc)
	require.NotNil(t, cfg.sqlOpenFunc)
	require.NotEqual(t, 0, cfg.connectMaxRetry)
	require.NotEqual(t, 0, cfg.connectRetryInterval)
	require.Equal(t, defaultConnMaxIdle, cfg.connMaxIdle)
	require.Equal(t, defaultConnMaxLifetime, cfg.connMaxLifetime)
	require.Equal(t, defaultConnMaxOpen, cfg.connMaxOpen)
}
