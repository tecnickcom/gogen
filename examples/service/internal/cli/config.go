package cli

import (
	"github.com/nexmoinc/gosrvlib/pkg/config"
	"github.com/nexmoinc/gosrvlib/pkg/validator"
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

	// fieldTagName is the name of the tag containing the original JSON field name
	fieldTagName = "mapstructure"
)

// ipifyConfig contains ipify client configuration
type ipifyConfig struct {
	Address string `mapstructure:"address" validate:"required,url"`
	Timeout int    `mapstructure:"timeout" validate:"required,min=1"`
}

// appConfig contains the full application configuration
type appConfig struct {
	config.BaseConfig `mapstructure:",squash" validate:"required"`
	Enabled           bool        `mapstructure:"enabled"`
	MonitoringAddress string      `mapstructure:"monitoring_address" validate:"required,hostname_port"`
	PublicAddress     string      `mapstructure:"public_address" validate:"required,hostname_port"`
	Ipify             ipifyConfig `mapstructure:"ipify" validate:"required"`
}

// SetDefaults sets the default configuration values in Viper
func (c *appConfig) SetDefaults(v config.Viper) {
	v.SetDefault("enabled", true)

	// Setting the default monitoring_address port to the same as service_port will start a single HTTP server
	v.SetDefault("monitoring_address", ":8072")
	v.SetDefault("public_address", ":8071")

	v.SetDefault("ipify.address", "https://api.ipify.org")
	v.SetDefault("ipify.timeout", 1)

	// NOTE: Set other configuration defaults here ...
}

// Validate performs the validation of the configuration values
func (c *appConfig) Validate() error {
	opts := []validator.Option{
		validator.WithFieldNameTag(fieldTagName),
		validator.WithErrorTemplates(validator.ErrorTemplates),
	}
	v, _ := validator.New(opts...)
	return v.ValidateStruct(c)
}
