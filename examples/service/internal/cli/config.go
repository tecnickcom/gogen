package cli

import (
	"fmt"

	"github.com/nexmoinc/gosrvlib/pkg/config"
)

const (
	// AppName is the name of the application executable
	AppName = "gosrvlibexample"

	// appEnvPrefix is the prefix of the configuration environment variables
	appEnvPrefix = "GOSRVLIBEXAMPLE"

	// appShortDesc is the short description of the application
	appShortDesc = "gosrvlibexampleshortdesc"

	// appLongDesc is the long description of the application
	appLongDesc = "gosrvlibexamplelongdesc"
)

// appConfig contains the full application configuration
type appConfig struct {
	config.BaseConfig `mapstructure:",squash"`
	MonitoringAddress string `mapstructure:"monitoring_address"`
	ServerAddress     string `mapstructure:"server_address"`
	Enabled           bool   `mapstructure:"enabled"`
}

// SetDefaults sets the default configuration values in Viper
func (c *appConfig) SetDefaults(v config.Viper) {
	v.SetDefault("enabled", true)

	// Setting the default monitoring_address port to the same as service_port will start a single HTTP server
	v.SetDefault("monitoring_address", ":8082")
	v.SetDefault("server_address", ":8081")

	// NOTE: Set other configuration defaults here
	// v.SetDefault("db.dsn", "<DSN>")
}

// Validate performs the validation of the configuration values
func (c *appConfig) Validate() error {
	if c.MonitoringAddress == "" {
		return fmt.Errorf("empty monitoring_address")
	}
	if c.ServerAddress == "" {
		return fmt.Errorf("empty server_address")
	}

	// NOTE: Implement validation for custom configuration options here
	return nil
}
