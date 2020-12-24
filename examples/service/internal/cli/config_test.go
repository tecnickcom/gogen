// +build unit

package cli

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func Test_appConfig_SetDefaults(t *testing.T) {
	t.Parallel()

	v := viper.New()
	c := &appConfig{}
	c.SetDefaults(v)

	require.True(t, v.GetBool("enabled"))
	require.NotEmpty(t, v.GetString("monitoring_address"))
	require.NotEmpty(t, v.GetString("public_address"))
	require.Equal(t, 5, len(v.AllKeys()))
}

func getValidTestConfig() appConfig {
	return appConfig{
		Enabled:           true,
		MonitoringAddress: ":1233",
		PublicAddress:     ":1231",
		Ipify: ipifyConfig{
			Address: "test.ipify.url.invalid",
			Timeout: 1,
		},
	}
}

func Test_appConfig_Validate(t *testing.T) {
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
			name:    "empty monitoring_address",
			fcfg:    func(cfg appConfig) appConfig { cfg.MonitoringAddress = ""; return cfg },
			wantErr: true,
		},
		{
			name:    "empty public_address",
			fcfg:    func(cfg appConfig) appConfig { cfg.PublicAddress = ""; return cfg },
			wantErr: true,
		},
		{
			name:    "empty ipify.address",
			fcfg:    func(cfg appConfig) appConfig { cfg.Ipify.Address = ""; return cfg },
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
