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

	require.NotEmpty(t, v.GetString("monitoring_address"))
	require.NotEmpty(t, v.GetString("server_address"))
	require.Equal(t, 2, len(v.AllKeys()))
}

func Test_appConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  appConfig
		wantErr bool
	}{
		{
			name: "empty monitoring_address",
			config: appConfig{
				MonitoringAddress: "",
			},
			wantErr: true,
		},
		{
			name: "empty server_address",
			config: appConfig{
				MonitoringAddress: "test",
				ServerAddress:     "",
			},
			wantErr: true,
		},
		{
			name: "valid config",
			config: appConfig{
				MonitoringAddress: "test",
				ServerAddress:     "test",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := tt.config.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
