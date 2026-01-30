package sqlconn

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_defaultConfig(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig("test_driver", "test_dsn")
	require.NotNil(t, cfg)
	require.Equal(t, "test_driver", cfg.driver)
	require.Equal(t, "test_dsn", cfg.dsn)
	require.NotNil(t, cfg.connectFunc)
	require.NotNil(t, cfg.checkConnectionFunc)
	require.NotNil(t, cfg.sqlOpenFunc)
	require.Equal(t, defaultConnMaxIdleCount, cfg.connMaxIdleCount)
	require.Equal(t, defaultConnMaxIdleTime, cfg.connMaxIdleTime)
	require.Equal(t, defaultConnMaxLifetime, cfg.connMaxLifetime)
	require.Equal(t, defaultConnMaxOpenCount, cfg.connMaxOpenCount)
	require.Equal(t, defaultPingTimeout, cfg.pingTimeout)
}

func Test_newConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		driver  string
		dsn     string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "fail with empty driver",
			driver:  "",
			dsn:     "dsnstring",
			opts:    []Option{},
			wantErr: true,
		},
		{
			name:    "fail with empty dsn",
			driver:  "driver",
			dsn:     "",
			opts:    []Option{},
			wantErr: true,
		},
		{
			name:   "success with empty options",
			driver: "driver",
			dsn:    "dsnstring",
			opts:   []Option{},
		},
		{
			name:   "success with driver from option",
			driver: "",
			dsn:    "dsnstring",
			opts:   []Option{WithDefaultDriver("driver")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := newConfig(tt.driver, tt.dsn, tt.opts...)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)
		})
	}
}

func Test_config_validate(t *testing.T) {
	t.Parallel()

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
			name: "fail with invalid max idle count",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.connMaxIdleCount = 0

				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "fail with invalid max idle time",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.connMaxIdleTime = 0

				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "fail with invalid max lifetime",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.connMaxLifetime = 0

				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "fail with invalid max open count",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.connMaxOpenCount = 0

				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "fail with invalid ping timeout",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.pingTimeout = 0

				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "fail with missing logger",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.logger = nil

				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "fail with missing shutdownWaitGroup",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.shutdownWaitGroup = nil

				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "fail with missing shutdownSignalChan",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.shutdownSignalChan = nil

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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.cfg.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
