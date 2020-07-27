//go:generate mockgen -package mocks -destination ../internal/mocks/config_mocks.go . Viper

package config

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	// nolint:golint
	_ "github.com/spf13/viper/remote"
)

const (
	defaultConfigName                = "config"
	defaultConfigType                = "json"
	defaultLogFormat                 = "JSON"
	defaultLogLevel                  = "INFO"
	defaultLogAddress                = ""
	defaultLogNetwork                = ""
	defaultRemoteConfigProvider      = ""
	defaultRemoteConfigEndpoint      = ""
	defaultRemoteConfigPath          = ""
	defaultRemoteConfigSecretKeyring = ""

	keyRemoteConfigProvider      = "remoteConfigProvider"
	keyRemoteConfigEndpoint      = "remoteConfigEndpoint"
	keyRemoteConfigPath          = "remoteConfigPath"
	keyRemoteConfigSecretKeyring = "remoteConfigSecretKeyring" // nolint:gosec
	keyRemoteConfigData          = "remoteConfigData"
	keyLogAddress                = "log.address"
	keyLogFormat                 = "log.format"
	keyLogLevel                  = "log.level"
	keyLogNetwork                = "log.network"

	providerEnvVar = "envvar"
)

// Configuration is the interface we need the application config struct to implement
type Configuration interface {
	SetDefaults(v Viper)
	Validate() error
}

// Viper is the local interface to the actual viper to allow for mocking
type Viper interface {
	AddConfigPath(in string)
	AddRemoteProvider(provider, endpoint, path string) error
	AddSecureRemoteProvider(provider, endpoint, path, secretkeyring string) error
	AllKeys() []string
	AutomaticEnv()
	BindEnv(input ...string) error
	BindPFlag(key string, flag *pflag.Flag) error
	Get(key string) interface{}
	ReadConfig(in io.Reader) error
	ReadInConfig() error
	ReadRemoteConfig() error
	SetConfigName(in string)
	SetConfigType(in string)
	SetDefault(key string, value interface{})
	SetEnvPrefix(in string)
	Unmarshal(rawVal interface{}, opts ...viper.DecoderConfigOption) error
}

// BaseConfig contains the default configuration options to be used in the application config struct
type BaseConfig struct {
	Log LogConfig `mapstructure:"log"`
}

// LogConfig contains the configuration for the application logger
type LogConfig struct {
	Level   string `mapstructure:"level"`   // Log level: EMERGENCY, ALERT, CRITICAL, ERROR, WARNING, NOTICE, INFO, DEBUG.
	Format  string `mapstructure:"format"`  // Log format: CONSOLE, JSON.
	Network string `mapstructure:"network"` // Network type used by the Syslog (i.e. udp or tcp).
	Address string `mapstructure:"address"` // Network address of the Syslog daemon (ip:port) or just (:port).
}

// remoteSourceConfig contains the default remote source options to be used in the application config struct
type remoteSourceConfig struct {
	Provider      string `mapstructure:"remoteConfigProvider"`      // remote configuration source ("consul", "etcd", "envvar")
	Endpoint      string `mapstructure:"remoteConfigEndpoint"`      // remote configuration URL (ip:port)
	Path          string `mapstructure:"remoteConfigPath"`          // remote configuration path where to search fo the configuration file ("/cli/numapidbsync")
	SecretKeyring string `mapstructure:"remoteConfigSecretKeyring"` // path to the openpgp secret keyring used to decript the remote configuration data ("/etc/numapidbsync/configkey.gpg")
	Data          string `mapstructure:"remoteConfigData"`          // base64 encoded JSON configuration data to be used with the "envvar" provider
}

var (
	localViper  Viper = viper.New()
	remoteViper Viper = viper.New()
)

// Load populates the configuration parameters
func Load(cmdName, configDir, envPrefix string, cfg Configuration) error {
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

// loadLocalConfig returns the local configuration parameters
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

	// add defaults from application configuration
	cfg.SetDefaults(v)

	// support environment variables for the remote configuration
	v.AutomaticEnv()
	v.SetEnvPrefix(strings.Replace(envPrefix, "-", "_", -1)) // will be uppercased automatically
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
		return nil, err
	}

	return &rsCfg, nil
}

// loadRemoteConfig returns the remote configuration parameters
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

func loadFromEnvVarSource(v Viper, rc *remoteSourceConfig, envPrefix string) error {
	if rc.Data == "" {
		return validationError(rc.Provider, envPrefix, keyRemoteConfigData)
	}

	data, err := base64.StdEncoding.DecodeString(rc.Data)
	if err != nil {
		return fmt.Errorf("failed decoding config data: %w", err)
	}

	return v.ReadConfig(bytes.NewReader(data))
}

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

	return v.ReadRemoteConfig()
}

func configureSearchPath(v Viper, cmdName, configDir string) {
	var configSearchPath []string

	// add cli dir from arguments
	if configDir != "" {
		configSearchPath = append(configSearchPath, configDir)
	}

	// add default cli search dirs
	configSearchPath = append(configSearchPath, []string{
		"../resources/test/etc/" + cmdName + "/",
		"./",
		"cli/",
		"$HOME/." + cmdName + "/",
		"/etc/" + cmdName + "/",
	}...)

	for _, p := range configSearchPath {
		v.AddConfigPath(p)
	}
}

func validationError(provider, envPrefix, varName string) error {
	return fmt.Errorf("%s config provider requires %s_%s to be set", provider, strings.ToUpper(envPrefix), strings.ToUpper(varName))
}
