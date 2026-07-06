package bootstrap

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_defaultConfig(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.context)
	require.Nil(t, cfg.logConfig)
	require.NotNil(t, cfg.createLoggerFunc)
	require.NotNil(t, cfg.createMetricsClientFunc)
}

func Test_defaultCreateLogger(t *testing.T) {
	t.Parallel()

	l := defaultCreateLogger()
	require.NotNil(t, l)
}

func Test_defaultCreateMetricsClientFunc(t *testing.T) {
	t.Parallel()

	m, err := defaultCreateMetricsClientFunc()
	require.NotNil(t, m)
	require.NoError(t, err)
}

func Test_config_validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		setupConfig func(c *config)
		wantErr     error
		name        string
	}{
		{
			name:        "succeed with default config",
			setupConfig: nil,
			wantErr:     nil,
		},
		{
			name: "fail with missing context",
			setupConfig: func(cfg *config) {
				cfg.context = nil
			},
			wantErr: ErrNilContext,
		},
		{
			name: "fail with missing createLoggerFunc",
			setupConfig: func(cfg *config) {
				cfg.createLoggerFunc = nil
			},
			wantErr: ErrNilCreateLoggerFunc,
		},
		{
			name: "fail with missing createMetricsClientFunc",
			setupConfig: func(cfg *config) {
				cfg.createMetricsClientFunc = nil
			},
			wantErr: ErrNilCreateMetricsClientFunc,
		},
		{
			name: "fail with missing shutdownWaitGroup",
			setupConfig: func(cfg *config) {
				cfg.shutdownWaitGroup = nil
			},
			wantErr: ErrNilShutdownWaitGroup,
		},
		{
			name: "fail with missing shutdownSignalChan",
			setupConfig: func(cfg *config) {
				cfg.shutdownSignalChan = nil
			},
			wantErr: ErrNilShutdownSignalChan,
		},
		{
			name: "fail with invalid shutdown timeout",
			setupConfig: func(cfg *config) {
				cfg.shutdownTimeout = 0
			},
			wantErr: ErrInvalidShutdownTimeout,
		},
		{
			name: "fail with nil logConfig set via WithLogConfig",
			setupConfig: func(cfg *config) {
				WithLogConfig(nil)(cfg)
			},
			wantErr: ErrNilLogConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := defaultConfig()
			if tt.setupConfig != nil {
				tt.setupConfig(cfg)
			}

			err := cfg.validate()
			if tt.wantErr == nil {
				require.NoError(t, err)

				return
			}

			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}
