package cli

import (
	"testing"

	"github.com/Vonage/gosrvlib/pkg/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func Test_appConfig_SetDefaults(t *testing.T) {
	t.Parallel()

	v := viper.New()
	c := &appConfig{}
	c.SetDefaults(v)

	require.True(t, v.GetBool("enabled"))
	require.Equal(t, 7, len(v.AllKeys()))
}

func getValidTestConfig() appConfig {
	return appConfig{
		BaseConfig: config.BaseConfig{
			Log: config.LogConfig{
				Level:   "DEBUG",
				Format:  "CONSOLE",
				Network: "tcp",
				Address: "127.0.0.1:1234",
			},
			ShutdownTimeout: 2,
		},
		Clients: cfgClients{
			Ipify: cfgClientIpify{
				Address: "https://test.ipify.url.invalid",
				Timeout: 13,
			},
		},
		Enabled: true,
		Servers: cfgServers{
			Monitoring: cfgServerMonitoring{
				Address: ":1233",
				Timeout: 11,
			},
			Public: cfgServerPublic{
				Address: ":1231",
				Timeout: 12,
			},
		},
	}
}

func Test_appConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		fcfg    func(cfg appConfig) appConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			fcfg:    func(cfg appConfig) appConfig { return cfg },
			wantErr: false,
		},
		{
			name:    "empty log.level",
			fcfg:    func(cfg appConfig) appConfig { cfg.Log.Level = ""; return cfg },
			wantErr: true,
		},
		{
			name:    "invalid log.level",
			fcfg:    func(cfg appConfig) appConfig { cfg.Log.Level = "WRONG_LOG_LEVEL"; return cfg },
			wantErr: true,
		},
		{
			name:    "empty log.format",
			fcfg:    func(cfg appConfig) appConfig { cfg.Log.Format = ""; return cfg },
			wantErr: true,
		},
		{
			name:    "invalid log.format",
			fcfg:    func(cfg appConfig) appConfig { cfg.Log.Format = "WRONG_LOG_FORMAT"; return cfg },
			wantErr: true,
		},
		{
			name:    "invalid log.network",
			fcfg:    func(cfg appConfig) appConfig { cfg.Log.Network = "WRONG_LOG_NETWORK"; return cfg },
			wantErr: true,
		},
		{
			name:    "invalid log.address",
			fcfg:    func(cfg appConfig) appConfig { cfg.Log.Address = "-WRONG_LOG_ADDRESS-"; return cfg },
			wantErr: true,
		},
		{
			name:    "invalid shutdown_timeout",
			fcfg:    func(cfg appConfig) appConfig { cfg.ShutdownTimeout = -1; return cfg },
			wantErr: true,
		},
		{
			name:    "empty servers",
			fcfg:    func(cfg appConfig) appConfig { cfg.Servers = cfgServers{}; return cfg },
			wantErr: true,
		},
		{
			name:    "empty servers.monitoring",
			fcfg:    func(cfg appConfig) appConfig { cfg.Servers.Monitoring = cfgServerMonitoring{}; return cfg },
			wantErr: true,
		},
		{
			name:    "empty servers.monitoring.address",
			fcfg:    func(cfg appConfig) appConfig { cfg.Servers.Monitoring.Address = ""; return cfg },
			wantErr: true,
		},
		{
			name: "invalid servers.monitoring.address",
			fcfg: func(cfg appConfig) appConfig {
				cfg.Servers.Monitoring.Address = "-WRONG_MONITORING_ADDRESS-"
				return cfg
			},
			wantErr: true,
		},
		{
			name:    "empty servers.monitoring.timeout",
			fcfg:    func(cfg appConfig) appConfig { cfg.Servers.Monitoring.Timeout = 0; return cfg },
			wantErr: true,
		},
		{
			name:    "empty servers.public",
			fcfg:    func(cfg appConfig) appConfig { cfg.Servers.Public = cfgServerPublic{}; return cfg },
			wantErr: true,
		},
		{
			name:    "empty servers.public.address",
			fcfg:    func(cfg appConfig) appConfig { cfg.Servers.Public.Address = ""; return cfg },
			wantErr: true,
		},
		{
			name:    "invalid servers.public.address",
			fcfg:    func(cfg appConfig) appConfig { cfg.Servers.Public.Address = "-WRONG_PUBLIC_ADDRESS-"; return cfg },
			wantErr: true,
		},
		{
			name:    "empty servers.public.timeout",
			fcfg:    func(cfg appConfig) appConfig { cfg.Servers.Public.Timeout = 0; return cfg },
			wantErr: true,
		},
		{
			name:    "empty clients",
			fcfg:    func(cfg appConfig) appConfig { cfg.Clients = cfgClients{}; return cfg },
			wantErr: true,
		},
		{
			name:    "empty clients.ipify",
			fcfg:    func(cfg appConfig) appConfig { cfg.Clients.Ipify = cfgClientIpify{}; return cfg },
			wantErr: true,
		},
		{
			name:    "empty clients.ipify.address",
			fcfg:    func(cfg appConfig) appConfig { cfg.Clients.Ipify.Address = ""; return cfg },
			wantErr: true,
		},
		{
			name:    "invalid clients.ipify.address",
			fcfg:    func(cfg appConfig) appConfig { cfg.Clients.Ipify.Address = "-WRONG_IPIFY_ADDRESS-"; return cfg },
			wantErr: true,
		},
		{
			name:    "empty clients.ipify.timeout",
			fcfg:    func(cfg appConfig) appConfig { cfg.Clients.Ipify.Timeout = 0; return cfg },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := tt.fcfg(getValidTestConfig())
			if err := cfg.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
