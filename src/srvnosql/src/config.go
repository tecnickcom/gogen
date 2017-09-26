package main

import (
	"errors"
	"strings"

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
	serverAddress string             // HTTP address (ip:port) or just (:port)
	log           *LogData           // Log level: EMERGENCY, ALERT, CRITICAL, ERROR, WARNING, NOTICE, INFO, DEBUG
	stats         *StatsData         // StatsD configuration, it is used to collect usage metrics
	mongodb       *MongodbData       // MongoDB configuration
	elasticsearch *ElasticsearchData // ElasticSearch configuration
}

var configDir string
var appParams = &params{}

// getConfigParams returns the configuration parameters
func getConfigParams() (par params, err error) {
	cfg, rcfg, err := getLocalConfigParams()
	if err != nil {
		return par, err
	}
	return getRemoteConfigParams(cfg, rcfg)
}

// getLocalConfigParams returns the local configuration parameters
func getLocalConfigParams() (cfg params, rcfg remoteConfigParams, err error) {

	viper.Reset()

	// set default remote configuration values
	viper.SetDefault("remoteConfigProvider", RemoteConfigProvider)
	viper.SetDefault("remoteConfigEndpoint", RemoteConfigEndpoint)
	viper.SetDefault("remoteConfigPath", RemoteConfigPath)
	viper.SetDefault("remoteConfigSecretKeyring", RemoteConfigSecretKeyring)

	// set default configuration values
	viper.SetDefault("serverAddress", ServerAddress)

	viper.SetDefault("log.level", LogLevel)
	viper.SetDefault("log.network", LogNetwork)
	viper.SetDefault("log.address", LogAddress)

	viper.SetDefault("stats.prefix", StatsPrefix)
	viper.SetDefault("stats.network", StatsNetwork)
	viper.SetDefault("stats.address", StatsAddress)
	viper.SetDefault("stats.flush_period", StatsFlushPeriod)

	viper.SetDefault("mongodb.address", MongodbAddress)
	viper.SetDefault("mongodb.database", MongodbDatabase)
	viper.SetDefault("mongodb.user", MongodbUser)
	viper.SetDefault("mongodb.password", MongodbPassword)
	viper.SetDefault("mongodb.timeout", MongodbTimeout)

	viper.SetDefault("elasticsearch.url", ElasticsearchURL)
	viper.SetDefault("elasticsearch.index", ElasticsearchIndex)
	viper.SetDefault("elasticsearch.username", ElasticsearchUsername)
	viper.SetDefault("elasticsearch.password", ElasticsearchPassword)

	// name of the configuration file without extension
	viper.SetConfigName("config")

	// configuration type
	viper.SetConfigType("json")

	if configDir != "" {
		viper.AddConfigPath(configDir)
	}

	// add local configuration paths
	for _, cpath := range ConfigPath {
		viper.AddConfigPath(cpath)
	}

	// Find and read the local configuration file (if any)
	err = viper.ReadInConfig()
	if err != nil {
		return cfg, rcfg, err
	}

	// read configuration parameters
	cfg = getViperParams()

	// support environment variables for the remote configuration
	viper.AutomaticEnv()
	viper.SetEnvPrefix(strings.Replace(ProgramName, "-", "_", -1)) // will be uppercased automatically
	envVar := []string{
		"remoteConfigProvider",
		"remoteConfigEndpoint",
		"remoteConfigPath",
		"remoteConfigSecretKeyring",
	}
	for _, ev := range envVar {
		err = viper.BindEnv(ev)
		if err != nil {
			return cfg, rcfg, err
		}
	}

	rcfg = remoteConfigParams{
		remoteConfigProvider:      viper.GetString("remoteConfigProvider"),
		remoteConfigEndpoint:      viper.GetString("remoteConfigEndpoint"),
		remoteConfigPath:          viper.GetString("remoteConfigPath"),
		remoteConfigSecretKeyring: viper.GetString("remoteConfigSecretKeyring"),
	}

	return cfg, rcfg, nil
}

// getRemoteConfigParams returns the remote configuration parameters
func getRemoteConfigParams(cfg params, rcfg remoteConfigParams) (params, error) {

	if rcfg.isEmpty() {
		return cfg, nil
	}

	viper.Reset()

	// set default configuration values

	viper.SetDefault("serverAddress", cfg.serverAddress)

	viper.SetDefault("log.level", cfg.log.Level)
	viper.SetDefault("log.network", cfg.log.Network)
	viper.SetDefault("log.address", cfg.log.Address)

	viper.SetDefault("stats.prefix", cfg.stats.Prefix)
	viper.SetDefault("stats.network", cfg.stats.Network)
	viper.SetDefault("stats.address", cfg.stats.Address)
	viper.SetDefault("stats.flush_period", cfg.stats.FlushPeriod)

	viper.SetDefault("mongodb.address", cfg.mongodb.Address)
	viper.SetDefault("mongodb.database", cfg.mongodb.Database)
	viper.SetDefault("mongodb.user", cfg.mongodb.User)
	viper.SetDefault("mongodb.password", cfg.mongodb.Password)
	viper.SetDefault("mongodb.timeout", cfg.mongodb.Timeout)

	viper.SetDefault("elasticsearch.url", cfg.elasticsearch.URL)
	viper.SetDefault("elasticsearch.index", cfg.elasticsearch.Index)
	viper.SetDefault("elasticsearch.username", cfg.elasticsearch.Username)
	viper.SetDefault("elasticsearch.password", cfg.elasticsearch.Password)

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
	return getViperParams(), nil
}

// getViperParams reads the config params via Viper
func getViperParams() params {
	return params{

		serverAddress: viper.GetString("serverAddress"),

		log: &LogData{
			Level:   viper.GetString("log.level"),
			Network: viper.GetString("log.network"),
			Address: viper.GetString("log.address"),
		},

		stats: &StatsData{
			Prefix:      viper.GetString("stats.prefix"),
			Network:     viper.GetString("stats.network"),
			Address:     viper.GetString("stats.address"),
			FlushPeriod: viper.GetInt("stats.flush_period"),
		},

		mongodb: &MongodbData{
			Address:  viper.GetString("mongodb.address"),
			Database: viper.GetString("mongodb.database"),
			User:     viper.GetString("mongodb.user"),
			Password: viper.GetString("mongodb.password"),
			Timeout:  viper.GetInt("mongodb.timeout"),
		},

		elasticsearch: &ElasticsearchData{
			URL:      viper.GetString("elasticsearch.url"),
			Index:    viper.GetString("elasticsearch.index"),
			Username: viper.GetString("elasticsearch.username"),
			Password: viper.GetString("elasticsearch.password"),
		},
	}
}

// checkParams cheks if the configuration parameters are valid
func checkParams(prm *params) error {
	// Log
	if prm.log.Level == "" {
		return errors.New("log.level is empty")
	}
	err := prm.log.setLog()
	if err != nil {
		return err
	}

	// Server
	if prm.serverAddress == "" {
		return errors.New("serverAddress is empty")
	}

	// StatsD
	if prm.stats.Prefix == "" {
		return errors.New("stats prefix is empty")
	}
	if prm.stats.Network != "udp" && prm.stats.Network != "tcp" {
		return errors.New("stats.network must be udp or tcp")
	}
	if prm.stats.FlushPeriod < 0 {
		return errors.New("stats.flush_period must be >= 0")
	}

	// MongoDB
	if prm.mongodb.Address == "" {
		return errors.New("mongodb.address is empty")
	}
	if prm.mongodb.Database == "" {
		return errors.New("mongodb.database is empty")
	}
	if prm.mongodb.Timeout < 1 {
		return errors.New("mongodb.timeout is empty")
	}

	// ElasticSearch
	if prm.elasticsearch.URL == "" {
		return errors.New("elasticsearch.url is empty")
	}
	if prm.elasticsearch.Index == "" {
		return errors.New("elasticsearch.index is empty")
	}

	return nil
}
