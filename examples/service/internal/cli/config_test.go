package cli

import (
	"testing"

	"github.com/nexmoinc/gosrvlib/pkg/config"
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
		},
		Enabled: true,
		Monitoring: serverConfig{
			Address: ":1233",
			Timeout: 1,
		},
		Public: serverConfig{
			Address: ":1231",
			Timeout: 1,
		},
		Ipify: ipifyConfig{
			Address: "https://test.ipify.url.invalid",
			Timeout: 1,
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
			name:    "empty monitoring.address",
			fcfg:    func(cfg appConfig) appConfig { cfg.Monitoring.Address = ""; return cfg },
			wantErr: true,
		},
		{
			name:    "invalid monitoring.address",
			fcfg:    func(cfg appConfig) appConfig { cfg.Monitoring.Address = "-WRONG_MONITORING_ADDRESS-"; return cfg },
			wantErr: true,
		},
		{
			name:    "empty monitoring.timeout",
			fcfg:    func(cfg appConfig) appConfig { cfg.Monitoring.Timeout = 0; return cfg },
			wantErr: true,
		},
		{
			name:    "empty public.address",
			fcfg:    func(cfg appConfig) appConfig { cfg.Public.Address = ""; return cfg },
			wantErr: true,
		},
		{
			name:    "invalid public.address",
			fcfg:    func(cfg appConfig) appConfig { cfg.Public.Address = "-WRONG_PUBLIC_ADDRESS-"; return cfg },
			wantErr: true,
		},
		{
			name:    "empty public.timeout",
			fcfg:    func(cfg appConfig) appConfig { cfg.Public.Timeout = 0; return cfg },
			wantErr: true,
		},
		{
			name:    "empty ipify.address",
			fcfg:    func(cfg appConfig) appConfig { cfg.Ipify.Address = ""; return cfg },
			wantErr: true,
		},
		{
			name:    "invalid ipify.address",
			fcfg:    func(cfg appConfig) appConfig { cfg.Ipify.Address = "-WRONG_IPIFY_ADDRESS-"; return cfg },
			wantErr: true,
		},
		{
			name:    "empty ipify.timeout",
			fcfg:    func(cfg appConfig) appConfig { cfg.Ipify.Timeout = 0; return cfg },
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
