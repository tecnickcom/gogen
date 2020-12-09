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

// ipifyConfig contains ipify client configuration
type ipifyConfig struct {
	Address string `mapstructure:"address"`
	Timeout int    `mapstructure:"timeout"`
}

// appConfig contains the full application configuration
type appConfig struct {
	config.BaseConfig `mapstructure:",squash"`
	Enabled           bool        `mapstructure:"enabled"`
	MonitoringAddress string      `mapstructure:"monitoring_address"`
	PublicAddress     string      `mapstructure:"public_address"`
	Ipify             ipifyConfig `mapstructure:"ipify"`
}

// SetDefaults sets the default configuration values in Viper
func (c *appConfig) SetDefaults(v config.Viper) {
	v.SetDefault("enabled", true)

	// Setting the default monitoring_address port to the same as service_port will start a single HTTP server
	v.SetDefault("monitoring_address", ":8072")
	v.SetDefault("public_address", ":8071")

	v.SetDefault("ipify.address", "https://api.ipify.org")
	v.SetDefault("ipify.timeout", 1)

	// NOTE: Set other configuration defaults here
	// v.SetDefault("db.dsn", "<DSN>")
}

// Validate performs the validation of the configuration values
func (c *appConfig) Validate() error {
	if c.MonitoringAddress == "" {
		return fmt.Errorf("empty monitoring_address")
	}
	if c.PublicAddress == "" {
		return fmt.Errorf("empty public_address")
	}
	if err := c.validateIpifyConfig(); err != nil {
		return err
	}

	// NOTE: Implement validation for custom configuration options here
	return nil
}

func (c *appConfig) validateIpifyConfig() error {
	if c.Ipify.Address == "" {
		return fmt.Errorf("empty ipify.address")
	}
	if c.Ipify.Timeout < 1 {
		return fmt.Errorf("ipify.timeout must be greater than 0")
	}
	return nil
}
