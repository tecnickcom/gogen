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
	require.NotNil(t, cfg.createLoggerFunc)
	require.NotNil(t, cfg.createMetricsClientFunc)
}

func Test_defaultCreateLogger(t *testing.T) {
	t.Parallel()

	l, err := defaultCreateLogger()
	require.NotNil(t, l)
	require.NoError(t, err)
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
		name        string
		setupConfig func(c *config)
		wantErr     bool
	}{
		{
			name: "fail with missing context",
			setupConfig: func(cfg *config) {
				cfg.context = nil
			},
			wantErr: true,
		},
		{
			name: "fail with missing createLoggerFunc",
			setupConfig: func(cfg *config) {
				cfg.createLoggerFunc = nil
			},
			wantErr: true,
		},
		{
			name: "fail with missing createMetricsClientFunc",
			setupConfig: func(cfg *config) {
				cfg.createMetricsClientFunc = nil
			},
			wantErr: true,
		},
		{
			name: "fail with missing shutdownWaitGroup",
			setupConfig: func(cfg *config) {
				cfg.shutdownWaitGroup = nil
			},
			wantErr: true,
		},
		{
			name: "fail with missing shutdownSignalChan",
			setupConfig: func(cfg *config) {
				cfg.shutdownSignalChan = nil
			},
			wantErr: true,
		},
		{
			name: "fail with invalid shutdown timeout",
			setupConfig: func(cfg *config) {
				cfg.shutdownTimeout = 0
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := defaultConfig()
			if tt.setupConfig != nil {
				tt.setupConfig(cfg)
			}

			if err := cfg.validate(); (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
