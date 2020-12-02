// +build unit

package httpserver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_defaultConfig(t *testing.T) {
	cfg := defaultConfig()
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.metricsHandlerFunc)
	require.NotNil(t, cfg.pingHandlerFunc)
	require.NotNil(t, cfg.pprofHandlerFunc)
	require.NotNil(t, cfg.statusHandlerFunc)
	require.NotNil(t, cfg.ipHandlerFunc)
	require.NotNil(t, cfg.router)
	require.NotEmpty(t, cfg.serverAddr)
	require.NotEqual(t, 0, cfg.shutdownTimeout)
	require.NotEmpty(t, cfg.traceIDHeaderName)
}

func Test_config_validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config
		wantErr bool
	}{
		{
			name: "fail with invalid httpServer address",
			cfg: func() *config {
				cfg := defaultConfig()
				cfg.serverAddr = "::"
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "fail with invalid shutdown timeout",
			cfg: func() *config {
				cfg := defaultConfig()
				cfg.shutdownTimeout = 0
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "fail with missing router",
			cfg: func() *config {
				cfg := defaultConfig()
				cfg.router = nil
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "fail with missing metrics handler",
			cfg: func() *config {
				cfg := defaultConfig()
				cfg.metricsHandlerFunc = nil
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "fail with missing ping handler",
			cfg: func() *config {
				cfg := defaultConfig()
				cfg.pingHandlerFunc = nil
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "fail with missing pprof handler",
			cfg: func() *config {
				cfg := defaultConfig()
				cfg.pprofHandlerFunc = nil
				return cfg
			}(),
			wantErr: true,
		}, {
			name: "fail with missing status handler",
			cfg: func() *config {
				cfg := defaultConfig()
				cfg.statusHandlerFunc = nil
				return cfg
			}(),
			wantErr: true,
		}, {
			name: "fail with missing ip handler",
			cfg: func() *config {
				cfg := defaultConfig()
				cfg.ipHandlerFunc = nil
				return cfg
			}(),
			wantErr: true,
		},
		{
			name:    "succeed with valid configuration",
			cfg:     defaultConfig(),
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

func Test_validateAddr(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{
			name:    "invalid empty address",
			addr:    "",
			wantErr: true,
		},
		{
			name:    "bad address",
			addr:    "::",
			wantErr: true,
		},
		{
			name:    "invalid unspecified port",
			addr:    ":",
			wantErr: true,
		},
		{
			name:    "invalid address port",
			addr:    ":aaa",
			wantErr: true,
		},
		{
			name:    "address port our of range",
			addr:    ":67800",
			wantErr: true,
		},
		{
			name:    "valid address (no host)",
			addr:    ":8017",
			wantErr: false,
		},
		{
			name:    "valid address (localhost)",
			addr:    "localhost:8017",
			wantErr: false,
		},
		{
			name:    "valid address (ip)",
			addr:    "0.0.0.0:8017",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateAddr(tt.addr)
			if tt.wantErr {
				require.NotNil(t, err, "validateAddr() addr = %q, error = %v, wantErr %v", tt.addr, err, tt.wantErr)
			} else {
				require.NoError(t, err, "validateAddr() addr = %q, error = %v, wantErr %v", tt.addr, err, tt.wantErr)
			}
		})
	}
}

func Test_config_isIndexRouteEnabled(t *testing.T) {
	tests := []struct {
		name                 string
		defaultEnabledRoutes []defaultRoute
		want                 bool
	}{
		{
			name:                 "should return true for enabled index route",
			defaultEnabledRoutes: []defaultRoute{IndexRoute, MetricsRoute},
			want:                 true,
		},
		{
			name:                 "should return false for enabled index route",
			defaultEnabledRoutes: []defaultRoute{MetricsRoute},
			want:                 false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := &config{
				defaultEnabledRoutes: tt.defaultEnabledRoutes,
			}
			if got := c.isIndexRouteEnabled(); got != tt.want {
				t.Errorf("isIndexRouteEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
