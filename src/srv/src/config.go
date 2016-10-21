package main

import (
	"errors"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote"
)

// params struct contains the application parameters
type remoteConfigParams struct {
	remoteConfigProvider      string // remote configuration source ("consul", "etcd")
	remoteConfigEndpoint      string // remote configuration URL (ip:port)
	remoteConfigPath          string // remote configuration path where to search fo the configuration file ("/config/~#PROJECT#~")
	remoteConfigSecretKeyring string // path to the openpgp secret keyring used to decript the remote configuration data ("/etc/~#PROJECT#~/configkey.gpg")
}

// isEmpty returns true if all the fields are empty strings
func (rcfg remoteConfigParams) isEmpty() bool {
	return rcfg.remoteConfigProvider == "" && rcfg.remoteConfigEndpoint == "" && rcfg.remoteConfigPath == "" && rcfg.remoteConfigSecretKeyring == ""
}

// params struct contains the application parameters
type params struct {
	serverAddress    string // HTTP API address for server mode (ip:port) or just (:port)
	statsPrefix      string // StatsD client's string prefix that will be used in every bucket name.
	statsNetwork     string // network type used by the StatsD client (i.e. udp or tcp).
	statsAddress     string // network address of the StatsD daemon (ip:port) or just (:port)
	statsFlushPeriod int    // How often (in milliseconds) the StatsD client's buffer is flushed.
	logLevel         string // Log level: NONE, EMERGENCY, ALERT, CRITICAL, ERROR, WARNING, NOTICE, INFO, DEBUG
}

var appParams = new(params)

// getConfigParams returns the configuration parameters
func getConfigParams() (params, error) {
	cfg, rcfg := getLocalConfigParams()
	return getRemoteConfigParams(cfg, rcfg)
}

// getLocalConfigParams returns the local configuration parameters
func getLocalConfigParams() (cfg params, rcfg remoteConfigParams) {

	viper.Reset()

	// set default remote configuration values
	viper.SetDefault("remoteConfigProvider", RemoteConfigProvider)
	viper.SetDefault("remoteConfigEndpoint", RemoteConfigEndpoint)
	viper.SetDefault("remoteConfigPath", RemoteConfigPath)
	viper.SetDefault("remoteConfigSecretKeyring", RemoteConfigSecretKeyring)

	// set default configuration values
	viper.SetDefault("serverAddress", ServerAddress)
	viper.SetDefault("statsPrefix", StatsPrefix)
	viper.SetDefault("statsNetwork", StatsNetwork)
	viper.SetDefault("statsAddress", StatsAddress)
	viper.SetDefault("statsFlushPeriod", StatsFlushPeriod)
	viper.SetDefault("logLevel", LogLevel)

	// name of the configuration file without extension
	viper.SetConfigName("config")

	// configuration type
	viper.SetConfigType("json")

	// add local configuration paths
	for _, cpath := range ConfigPath {
		viper.AddConfigPath(cpath)
	}

	// Find and read the local configuration file (if any)
	viper.ReadInConfig()

	// read configuration parameters
	cfg = params{
		serverAddress:    viper.GetString("serverAddress"),
		statsPrefix:      viper.GetString("statsPrefix"),
		statsNetwork:     viper.GetString("statsNetwork"),
		statsAddress:     viper.GetString("statsAddress"),
		statsFlushPeriod: viper.GetInt("statsFlushPeriod"),
		logLevel:         viper.GetString("logLevel"),
	}

	// support environment variables for the remote configuration
	viper.AutomaticEnv()
	viper.SetEnvPrefix(strings.Replace(ProgramName, "-", "_", -1)) // will be uppercased automatically
	viper.BindEnv("remoteConfigProvider")
	viper.BindEnv("remoteConfigEndpoint")
	viper.BindEnv("remoteConfigPath")
	viper.BindEnv("remoteConfigSecretKeyring")

	rcfg = remoteConfigParams{
		remoteConfigProvider:      viper.GetString("remoteConfigProvider"),
		remoteConfigEndpoint:      viper.GetString("remoteConfigEndpoint"),
		remoteConfigPath:          viper.GetString("remoteConfigPath"),
		remoteConfigSecretKeyring: viper.GetString("remoteConfigSecretKeyring"),
	}

	return cfg, rcfg
}

// getRemoteConfigParams returns the remote configuration parameters
func getRemoteConfigParams(cfg params, rcfg remoteConfigParams) (params, error) {

	if rcfg.isEmpty() {
		return cfg, nil
	}

	viper.Reset()

	// set default configuration values
	viper.SetDefault("serverAddress", cfg.serverAddress)
	viper.SetDefault("statsPrefix", cfg.statsPrefix)
	viper.SetDefault("statsNetwork", cfg.statsNetwork)
	viper.SetDefault("statsAddress", cfg.statsAddress)
	viper.SetDefault("statsFlushPeriod", cfg.statsFlushPeriod)
	viper.SetDefault("logLevel", cfg.logLevel)

	// configuration type
	viper.SetConfigType("json")

	// add remote configuration provider
	var err error
	if rcfg.remoteConfigSecretKeyring == "" {
		err = viper.AddRemoteProvider(rcfg.remoteConfigProvider, rcfg.remoteConfigEndpoint, rcfg.remoteConfigPath)
	} else {
		err = viper.AddSecureRemoteProvider(rcfg.remoteConfigProvider, rcfg.remoteConfigEndpoint, rcfg.remoteConfigPath, rcfg.remoteConfigSecretKeyring)
	}
	if err == nil {
		// try to read the remote configuration (if any)
		err = viper.ReadRemoteConfig()
	}
	if err != nil {
		return cfg, err
	}

	// read configuration parameters
	return params{
			serverAddress:    viper.GetString("serverAddress"),
			statsPrefix:      viper.GetString("statsPrefix"),
			statsNetwork:     viper.GetString("statsNetwork"),
			statsAddress:     viper.GetString("statsAddress"),
			statsFlushPeriod: viper.GetInt("statsFlushPeriod"),
			logLevel:         viper.GetString("logLevel"),
		},
		nil
}

// checkParams cheks if the configuration parameters are valid
func checkParams(prm *params) error {
	if prm.serverAddress == "" {
		return errors.New("The Server address is empty")
	}
	if prm.statsNetwork != "udp" && prm.statsNetwork != "tcp" {
		return errors.New("The statsNetwork must be udp or tcp")
	}
	if prm.statsFlushPeriod < 0 {
		return errors.New("The statsFlushPeriod must be >= 0")
	}
	if prm.logLevel == "" {
		return errors.New("logLevel is empty")
	}
	levelCode, err := log.ParseLevel(prm.logLevel)
	if err != nil {
		return errors.New("The logLevel must be one of the following: panic, fatal, error, warning, info, debug")
	}
	log.SetLevel(levelCode)
	return nil
}
