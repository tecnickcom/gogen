package sqlconn

import (
	"strings"
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
	require.Equal(t, defaultValidationQuery, cfg.validationQuery)
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
		{
			name:   "success with whitespace-padded driver and dsn trimmed",
			driver: "  driver  ",
			dsn:    "\tdsnstring\n",
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

			// When no option overrides the driver, the stored driver/dsn must be
			// the trimmed form of the inputs.
			if len(tt.opts) == 0 {
				require.Equal(t, strings.TrimSpace(tt.driver), cfg.driver)
				require.Equal(t, strings.TrimSpace(tt.dsn), cfg.dsn)
			}
		})
	}
}

func Test_config_validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cfg       *config
		wantErr   bool
		wantErrIs error
	}{
		{
			name:      "fail with empty driver",
			cfg:       defaultConfig("", "user:pass@tcp(127.0.0.1:1234)/testdb"),
			wantErr:   true,
			wantErrIs: ErrDriverRequired,
		},
		{
			name:      "fail with empty DSN",
			cfg:       defaultConfig("sqldb", ""),
			wantErr:   true,
			wantErrIs: ErrDSNRequired,
		},
		{
			name: "fail with invalid connect function",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.connectFunc = nil

				return cfg
			}(),
			wantErr:   true,
			wantErrIs: ErrNilConnectFunc,
		},
		{
			name: "fail with invalid check connection function",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.checkConnectionFunc = nil

				return cfg
			}(),
			wantErr:   true,
			wantErrIs: ErrNilCheckConnectionFunc,
		},
		{
			name: "fail with invalid sql open function",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.sqlOpenFunc = nil

				return cfg
			}(),
			wantErr:   true,
			wantErrIs: ErrNilSQLOpenFunc,
		},
		{
			name: "fail with negative max idle count",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.connMaxIdleCount = -1

				return cfg
			}(),
			wantErr:   true,
			wantErrIs: ErrInvalidMaxIdleCount,
		},
		{
			name: "fail with negative max idle time",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.connMaxIdleTime = -1

				return cfg
			}(),
			wantErr:   true,
			wantErrIs: ErrInvalidMaxIdleTime,
		},
		{
			name: "fail with negative max lifetime",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.connMaxLifetime = -1

				return cfg
			}(),
			wantErr:   true,
			wantErrIs: ErrInvalidMaxLifetime,
		},
		{
			name: "fail with negative max open count",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.connMaxOpenCount = -1

				return cfg
			}(),
			wantErr:   true,
			wantErrIs: ErrInvalidMaxOpenCount,
		},
		{
			name: "fail with invalid ping timeout",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.pingTimeout = 0

				return cfg
			}(),
			wantErr:   true,
			wantErrIs: ErrInvalidPingTimeout,
		},
		{
			name: "fail with empty validation query",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.validationQuery = "   "

				return cfg
			}(),
			wantErr:   true,
			wantErrIs: ErrEmptyValidationQuery,
		},
		{
			name: "fail with missing logger",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.logger = nil

				return cfg
			}(),
			wantErr:   true,
			wantErrIs: ErrNilLogger,
		},
		{
			name: "fail with missing shutdownWaitGroup",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.shutdownWaitGroup = nil

				return cfg
			}(),
			wantErr:   true,
			wantErrIs: ErrNilShutdownWaitGroup,
		},
		{
			name: "fail with missing shutdownSignalChan",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.shutdownSignalChan = nil

				return cfg
			}(),
			wantErr:   true,
			wantErrIs: ErrNilShutdownSignalChan,
		},
		{
			name: "succeed with no errors",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				return cfg
			}(),
			wantErr: false,
		},
		{
			name: "succeed with zero pool limits (unlimited/disabled)",
			cfg: func() *config {
				cfg := defaultConfig("sqldb", "user:pass@tcp(127.0.0.1:1234)/testdb")
				cfg.connMaxIdleCount = 0
				cfg.connMaxIdleTime = 0
				cfg.connMaxLifetime = 0
				cfg.connMaxOpenCount = 0

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

			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
			}
		})
	}
}
