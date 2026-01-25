package cli

import (
	"github.com/tecnickcom/gogen/pkg/config"
	"github.com/tecnickcom/gogen/pkg/validator"
)

const (
	// AppName is the name of the application executable.
	AppName = "gogenexample"

	// appEnvPrefix is the prefix of the configuration environment variables.
	appEnvPrefix = "GOGENEXAMPLE"

	// appShortDesc is the short description of the application.
	appShortDesc = "gogenexampleshortdesc"

	// appLongDesc is the long description of the application.
	appLongDesc = "gogenexamplelongdesc"

	// fieldTagName is the name of the tag containing the original JSON field name.
	fieldTagName = "mapstructure"
)

type cfgServer struct {
	Address string `mapstructure:"address" validate:"required,hostname_port"`
	Timeout int    `mapstructure:"timeout" validate:"required,min=1"`
}

type cfgServerMonitoring cfgServer

type cfgServerPrivate cfgServer

type cfgServerPublic cfgServer

// cfgServers contains the configuration for all exposed servers.
type cfgServers struct {
	Monitoring cfgServerMonitoring `mapstructure:"monitoring" validate:"required"`
	Private    cfgServerPrivate    `mapstructure:"private"    validate:"required"`
	Public     cfgServerPublic     `mapstructure:"public"     validate:"required"`
}

type cfgClientIpify struct {
	Address string `mapstructure:"address" validate:"required,url"`
	Timeout int    `mapstructure:"timeout" validate:"required,min=1"`
}

// cfgClients contains the configuration for all external clients.
type cfgClients struct {
	Ipify cfgClientIpify `mapstructure:"ipify" validate:"required"`
}

type cfgDB struct {
	// Database in DSN format: "username:password@protocol(address)/dbname?param=value"
	DSN              string `mapstructure:"dsn"                 validate:"required"`
	TimeoutPing      int    `mapstructure:"timeout_ping"        validate:"required"`
	ConnMaxOpen      int    `mapstructure:"conn_max_open"       validate:"required,min=0"`
	ConnMaxIdleCount int    `mapstructure:"conn_max_idle_count" validate:"required,min=0"`
	ConnMaxIdleTime  int    `mapstructure:"conn_max_idle_time"  validate:"required,min=0"`
	ConnMaxLifetime  int    `mapstructure:"conn_max_lifetime"   validate:"required,min=0"`
}

type cfgDatabases struct {
	Enabled bool  `mapstructure:"enabled"`
	Main    cfgDB `mapstructure:"main"    validate:"required_if=Enabled true"`
	Read    cfgDB `mapstructure:"read"    validate:"required_if=Enabled true"`
}

// appConfig contains the full application configuration.
type appConfig struct {
	config.BaseConfig `mapstructure:",squash" validate:"required"`

	Enabled bool         `mapstructure:"enabled"`
	Servers cfgServers   `mapstructure:"servers" validate:"required"`
	Clients cfgClients   `mapstructure:"clients" validate:"required"`
	DB      cfgDatabases `mapstructure:"db"      validate:"required"`
}

// SetDefaults sets the default configuration values in Viper.
func (c *appConfig) SetDefaults(v config.Viper) {
	v.SetDefault("enabled", true)

	v.SetDefault("servers.monitoring.address", ":8071")
	v.SetDefault("servers.monitoring.timeout", 60)

	v.SetDefault("servers.private.address", ":8072")
	v.SetDefault("servers.private.timeout", 60)

	v.SetDefault("servers.public.address", ":8073")
	v.SetDefault("servers.public.timeout", 60)

	v.SetDefault("clients.ipify.address", "https://api.ipify.org")
	v.SetDefault("clients.ipify.timeout", 1)

	v.SetDefault("db.enabled", false)

	v.SetDefault("db.main.conn_max_idle_count", 5)
	v.SetDefault("db.main.conn_max_idle_time", 60)
	v.SetDefault("db.main.conn_max_lifetime", 3600)
	v.SetDefault("db.main.conn_max_open", 50)
	v.SetDefault("db.main.dsn", "")
	v.SetDefault("db.main.timeout_ping", 1)

	v.SetDefault("db.read.conn_max_idle_count", 5)
	v.SetDefault("db.read.conn_max_idle_time", 60)
	v.SetDefault("db.read.conn_max_lifetime", 3600)
	v.SetDefault("db.read.conn_max_open", 50)
	v.SetDefault("db.read.dsn", "")
	v.SetDefault("db.read.timeout_ping", 1)
}

// Validate performs the validation of the configuration values.
func (c *appConfig) Validate() error {
	opts := []validator.Option{
		validator.WithFieldNameTag(fieldTagName),
		validator.WithCustomValidationTags(validator.CustomValidationTags()),
		validator.WithErrorTemplates(validator.ErrorTemplates()),
	}

	v, _ := validator.New(opts...)

	return v.ValidateStruct(c) //nolint:wrapcheck
}
