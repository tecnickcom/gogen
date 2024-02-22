/*
Package config handles the configuration of a program.
The configuration contains the set of initial parameter settings that are read at runtime by the program.

This package allows plugging a fully-fledged configuration system into an application, taking care of the boilerplate code, common settings, configuration loading, and validation.

Different configuration sources can be used during development, debugging, testing, or deployment.
The configuration can be loaded from a local file, environment variables, or a remote configuration provider (e.g., Consul, etcd, etcd3, Firestore, NATS).

This is a Viper-based implementation of the configuration model described in the following article:
  - Nicola Asuni, 2014-09-13, "Software Configuration", https://technick.net/guides/software/software_configuration/

# Configuration Loading Strategy:

To achieve maximum flexibility, the different configuration entry points are coordinated in the following sequence (1 has the lowest priority and 5 has the highest):

 1. In the "myprog" program, the configuration parameters are defined as a data structure that can be easily mapped to and from a JSON object (or any other format supported by Viper like TOML, YAML, and HCL).
    Each structure parameter is annotated with the "mapstructure" and "validate" tags to define the name mapping and the validation rules.
    The parameters are initialized with constant default values.

 2. The program attempts to load the local "config.json" configuration file, and as soon as one is found, it overwrites the default values previously set.

    The configuration file is searched in the following ordered directories based on the Linux Filesystem Hierarchy Standard (FHS):

    - ./

    - ~/.myprog/

    - /etc/myprog/

 3. The program attempts to load the environmental variables that define the remote configuration system. If found, it overwrites the corresponding configuration parameters:

    - [ENVIRONMENT VARIABLE NAME] → [CONFIGURATION PARAMETER NAME]

    - MYPROG_REMOTECONFIGPROVIDER → remoteConfigProvider

    - MYPROG_REMOTECONFIGENDPOINT → remoteConfigEndpoint

    - MYPROG_REMOTECONFIGPATH → remoteConfigPath

    - MYPROG_REMOTECONFIGSECRETKEYRING → remoteConfigSecretKeyring

    - MYPROG_REMOTECONFIGDATA → remoteConfigData

 4. If the "remoteConfigProvider" parameter is not empty, the program attempts to load the configuration data from the specified source.
    This can be any remote source supported by the Viper library (e.g., Consul, etcd, etcd3, Firestore, NATS).
    The configuration source can also be the "MYPROG_REMOTECONFIGDATA" environment variable as base64-encoded JSON when "MYPROG_REMOTECONFIGPROVIDER" is set to "envvar".

 5. Any specified command-line argument overwrites the corresponding configuration parameter.

 6. The configuration parameters are validated via the Validate() function.

# Example:

  - An implementation example of this configuration package can be found in examples/service/internal/cli/config.go
    Note that the "log", "shutdown_timeout", and "remoteConfig" parameters are defined in this package as they are common to all programs.

  - The configuration file format of the example service is defined by examples/service/resources/etc/gosrvlibexample/config.schema.json

  - The default configuration file of the example service is defined by examples/service/resources/etc/gosrvlibexample/config.json
*/
package config

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote" //nolint:revive,nolintlint
)

// General constants.
const (
	defaultConfigName = "config" // Base name of the file containing the local configuration data.
	defaultConfigType = "json"   // Type and file extension of the file containing the local configuration data.
	providerEnvVar    = "envvar" // Provider name for the environment variable configuration source.
)

// Remote configuration key names.
const (
	keyRemoteConfigProvider      = "remoteConfigProvider"
	keyRemoteConfigEndpoint      = "remoteConfigEndpoint"
	keyRemoteConfigPath          = "remoteConfigPath"
	keyRemoteConfigSecretKeyring = "remoteConfigSecretKeyring" //nolint:gosec
	keyRemoteConfigData          = "remoteConfigData"
)

// Remote configuration default values.
const (
	defaultRemoteConfigProvider      = ""
	defaultRemoteConfigEndpoint      = ""
	defaultRemoteConfigPath          = ""
	defaultRemoteConfigSecretKeyring = ""
)

// Logger configuration key names.
const (
	keyLogAddress = "log.address"
	keyLogFormat  = "log.format"
	keyLogLevel   = "log.level"
	keyLogNetwork = "log.network"
)

// Logger configuration default values.
const (
	defaultLogFormat  = "JSON"
	defaultLogLevel   = "DEBUG"
	defaultLogAddress = ""
	defaultLogNetwork = ""
)

// Extra parameters key names.
const (
	keyShutdownTimeout = "shutdown_timeout"
)

// Extra parameters default values.
const (
	defaultShutdownTimeout = 30 // time in seconds to wait on exit for a graceful shutdown.
)

// Configuration is the interface we need the application config struct to implement.
type Configuration interface {
	SetDefaults(v Viper)
	Validate() error
}

// Viper is the local interface to the actual viper to allow for mocking.
//
//nolint:interfacebloat
type Viper interface {
	AddConfigPath(in string)
	AddRemoteProvider(provider, endpoint, path string) error
	AddSecureRemoteProvider(provider, endpoint, path, secretkeyring string) error
	AllKeys() []string
	AutomaticEnv()
	BindEnv(input ...string) error
	BindPFlag(key string, flag *pflag.Flag) error
	Get(key string) any
	ReadConfig(in io.Reader) error
	ReadInConfig() error
	ReadRemoteConfig() error
	SetConfigName(in string)
	SetConfigType(in string)
	SetDefault(key string, value any)
	SetEnvPrefix(in string)
	Unmarshal(rawVal any, opts ...viper.DecoderConfigOption) error
}

// BaseConfig contains the default configuration options to be used in the application config struct.
type BaseConfig struct {
	// Log configuration.
	Log LogConfig `mapstructure:"log" validate:"required"`

	// ShutdownTimeout is the time in seconds to wait for graceful shutdown.
	ShutdownTimeout int64 `mapstructure:"shutdown_timeout" validate:"omitempty,min=1,max=3600"`
}

// LogConfig contains the configuration for the application logger.
type LogConfig struct {
	// Level is the standard syslog level: EMERGENCY, ALERT, CRITICAL, ERROR, WARNING, NOTICE, INFO, DEBUG.
	Level string `mapstructure:"level" validate:"required,oneof=EMERGENCY ALERT CRITICAL ERROR WARNING NOTICE INFO DEBUG"`

	// Format is the log output format: CONSOLE, JSON.
	Format string `mapstructure:"format" validate:"required,oneof=CONSOLE JSON"`

	// Network is the optional network protocol used to send logs via syslog: udp, tcp.
	Network string `mapstructure:"network" validate:"omitempty,oneof=udp tcp"`

	// Address is the optional remote syslog network address: (ip:port) or just (:port).
	Address string `mapstructure:"address" validate:"omitempty,hostname_port"`
}

// remoteSourceConfig contains the default remote source options to be used in the application config struct.
type remoteSourceConfig struct {
	// Provider is the optional external configuration source: consul, envvar, etcd, etcd3, firestore, nats.
	// When envvar is set the data should be set in the Data field.
	Provider string `mapstructure:"remoteConfigProvider" validate:"omitempty,oneof=consul envvar etcd etcd3 firestore nats"`

	// Endpoint is the remote configuration URL (ip:port).
	Endpoint string `mapstructure:"remoteConfigEndpoint" validate:"omitempty,url|hostname_port"`

	// Path is the remote configuration path where to search fo the configuration file ("/cli/program").
	Path string `mapstructure:"remoteConfigPath" validate:"omitempty,file"`

	// SecretKeyring is the path to the openpgp secret keyring used to decript the remote configuration data (e.g.: "/etc/program/configkey.gpg")
	SecretKeyring string `mapstructure:"remoteConfigSecretKeyring" validate:"omitempty,file"`

	// Data is the base64 encoded JSON configuration data to be used with the "envvar" provider.
	Data string `mapstructure:"remoteConfigData" validate:"required_if=Provider envar,omitempty,base64"`
}

// Load populates the configuration parameters.
func Load(cmdName, configDir, envPrefix string, cfg Configuration) error {
	localViper := viper.New()
	remoteViper := viper.New()

	return loadConfig(localViper, remoteViper, cmdName, configDir, envPrefix, cfg)
}

// loadConfig loads the configuration.
func loadConfig(localViper, remoteViper Viper, cmdName, configDir, envPrefix string, cfg Configuration) error {
	remoteSourceCfg, err := loadLocalConfig(localViper, cmdName, configDir, envPrefix, cfg)
	if err != nil {
		return fmt.Errorf("failed loading local configuration: %w", err)
	}

	if err := loadRemoteConfig(localViper, remoteViper, remoteSourceCfg, envPrefix, cfg); err != nil {
		return fmt.Errorf("failed loading remote configuration: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("failed validating configuration: %w", err)
	}

	return nil
}

// loadLocalConfig returns the local configuration parameters.
func loadLocalConfig(v Viper, cmdName, configDir, envPrefix string, cfg Configuration) (*remoteSourceConfig, error) {
	// set default remote configuration values
	v.SetDefault(keyRemoteConfigProvider, defaultRemoteConfigProvider)
	v.SetDefault(keyRemoteConfigEndpoint, defaultRemoteConfigEndpoint)
	v.SetDefault(keyRemoteConfigPath, defaultRemoteConfigPath)
	v.SetDefault(keyRemoteConfigSecretKeyring, defaultRemoteConfigSecretKeyring)

	// set default logging configuration values
	v.SetDefault(keyLogFormat, defaultLogFormat)
	v.SetDefault(keyLogLevel, defaultLogLevel)
	v.SetDefault(keyLogAddress, defaultLogAddress)
	v.SetDefault(keyLogNetwork, defaultLogNetwork)

	// set default config name and type
	v.SetConfigName(defaultConfigName)
	v.SetConfigType(defaultConfigType)

	// add default search paths
	configureSearchPath(v, cmdName, configDir)

	// set application defaults
	v.SetDefault(keyShutdownTimeout, defaultShutdownTimeout)

	// set defaults from application configuration
	cfg.SetDefaults(v)

	// support environment variables for the remote configuration
	v.AutomaticEnv()
	v.SetEnvPrefix(strings.ReplaceAll(envPrefix, "-", "_")) // will be uppercased automatically

	envVar := []string{
		keyRemoteConfigProvider,
		keyRemoteConfigEndpoint,
		keyRemoteConfigPath,
		keyRemoteConfigSecretKeyring,
		keyRemoteConfigData,
	}

	for _, ev := range envVar {
		_ = v.BindEnv(ev) // we ignore the error because we are always passing an argument value
	}

	// Find and read the local configuration file (if any)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed reading in config: %w", err)
	}

	var rsCfg remoteSourceConfig

	if err := v.Unmarshal(&rsCfg); err != nil {
		return nil, fmt.Errorf("failed unmarshalling config: %w", err)
	}

	return &rsCfg, nil
}

// loadRemoteConfig returns the remote configuration parameters.
func loadRemoteConfig(lv Viper, rv Viper, rs *remoteSourceConfig, envPrefix string, cfg Configuration) error {
	for _, k := range lv.AllKeys() {
		rv.SetDefault(k, lv.Get(k))
	}

	rv.SetConfigType(defaultConfigType)

	var err error

	switch rs.Provider {
	case "":
		// ignore remote source
	case providerEnvVar:
		err = loadFromEnvVarSource(rv, rs, envPrefix)
	default:
		err = loadFromRemoteSource(rv, rs, envPrefix)
	}

	if err != nil {
		return fmt.Errorf("failed loading configuration from remote source: %w", err)
	}

	if err := rv.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed loading application configuration: %w", err)
	}

	return nil
}

// loadFromEnvVarSource loads the configuration data from an environment variable.
// The data must be base64-encoded.
func loadFromEnvVarSource(v Viper, rc *remoteSourceConfig, envPrefix string) error {
	if rc.Data == "" {
		return validationError(rc.Provider, envPrefix, keyRemoteConfigData)
	}

	data, err := base64.StdEncoding.DecodeString(rc.Data)
	if err != nil {
		return fmt.Errorf("failed decoding config data: %w", err)
	}

	return v.ReadConfig(bytes.NewReader(data)) //nolint:wrapcheck
}

// loadFromRemoteSource loads the configuration data from a remote source or service.
func loadFromRemoteSource(v Viper, rc *remoteSourceConfig, envPrefix string) error {
	if rc.Endpoint == "" {
		return validationError(rc.Provider, envPrefix, keyRemoteConfigEndpoint)
	}

	if rc.Path == "" {
		return validationError(rc.Provider, envPrefix, keyRemoteConfigPath)
	}

	var err error

	if rc.SecretKeyring == "" {
		err = v.AddRemoteProvider(rc.Provider, rc.Endpoint, rc.Path)
	} else {
		err = v.AddSecureRemoteProvider(rc.Provider, rc.Endpoint, rc.Path, rc.SecretKeyring)
	}

	if err != nil {
		return fmt.Errorf("failed adding remote config provider: %w", err)
	}

	return v.ReadRemoteConfig() //nolint:wrapcheck
}

// configureSearchPath sets the directory paths to search in order for a local configuration file.
func configureSearchPath(v Viper, cmdName, configDir string) {
	var configSearchPath []string

	if configDir != "" {
		// add the configuration directory specified as program argument
		configSearchPath = append(configSearchPath, configDir)
	}

	// add default search directories for the configuration file
	configSearchPath = append(configSearchPath, []string{
		"./",
		"$HOME/." + cmdName + "/",
		"/etc/" + cmdName + "/",
	}...)

	for _, p := range configSearchPath {
		v.AddConfigPath(p)
	}
}

// validationError returns a validation error.
func validationError(provider, envPrefix, varName string) error {
	return fmt.Errorf("%s config provider requires %s_%s to be set", provider, strings.ToUpper(envPrefix), strings.ToUpper(varName))
}
