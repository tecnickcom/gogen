package main

// ProgramName defines this application name
const ProgramName = "~#PROJECT#~"

// ProgramVersion set this application version
// This is supposed to be automatically populated by the Makefile using the value from the VERSION file
// (-ldflags '-X main.ProgramVersion=$(shell cat VERSION)')
var ProgramVersion = "0.0.0"

// ProgramRelease contains this program release number (or build number)
// This is automatically populated by the Makefile using the value from the RELEASE file
// (-ldflags '-X main.ProgramRelease=$(shell cat RELEASE)')
var ProgramRelease = "0"

// ConfigPath list the paths where to look for configuration files (in order)
var ConfigPath = [...]string{
	"../resources/test/etc/" + ProgramName + "/",
	"./",
	"config/",
	"$HOME/." + ProgramName + "/",
	"/etc/" + ProgramName + "/",
}

// RemoteConfigProvider is the remote configuration source ("consul", "etcd")
const RemoteConfigProvider = ""

// RemoteConfigEndpoint is the remote configuration URL (ip:port)
const RemoteConfigEndpoint = ""

// RemoteConfigPath is the remote configuration path where to search fo the configuration file ("/config/~#PROJECT#~")
const RemoteConfigPath = ""

// RemoteConfigSecretKeyring is the path to the openpgp secret keyring used to decript the remote configuration data ("/etc/~#PROJECT#~/configkey.gpg")
const RemoteConfigSecretKeyring = "" // #nosec

// ServerAddress is the default HTTP address (ip:port) or just (:port)
const ServerAddress = ":8081"

// ServerShutdownTimeout timeout in seconds before forcing the server to close
const ServerShutdownTimeout = 10

// Log (syslog)

// LogLevel defines the default log level: EMERGENCY, ALERT, CRITICAL, ERROR, WARNING, NOTICE, INFO, DEBUG
const LogLevel = "INFO"

// LogNetwork is the network type used by Syslog (i.e. udp or tcp). Leave emty to disable.
const LogNetwork = ""

// LogAddress is the network address of the Syslog daemon (ip:port) or just (:port). Leave emty to disable.
const LogAddress = ""

// StatsD is used to collect usage metrics

// StatsPrefix is the StatsD client's string prefix that will be used in every bucket name.
const StatsPrefix = "~#PROJECT#~"

// StatsNetwork is the network type used by the StatsD client (i.e. udp or tcp).
const StatsNetwork = "udp"

// StatsAddress is the network address of the StatsD daemon (ip:port) or just (:port)
const StatsAddress = ":8125"

// StatsFlushPeriod sets how often (in milliseconds) the StatsD client's buffer is flushed.
// When 0 the buffer is only flushed when it is full.
const StatsFlushPeriod = 100

// ----------

// Network address of the ElasticSearch serice (ip:port) or just (:port)
const ElasticsearchURL = "http://127.0.0.1:9200"

// ElasticSearch main index
const ElasticsearchIndex = ""

// ElasticSearch user name
const ElasticsearchUsername = ""

// ElasticSearch password
const ElasticsearchPassword = ""

// ----------

// Network address of the MongoDB serice (ip:port) or just (:port)
const MongodbAddress = "mongodb://127.0.0.1:27017"

// MongoDB Database name
const MongodbDatabase = "test"

// MongoDB user name
const MongodbUser = ""

// MongoDB password
const MongodbPassword = ""

// MongoDB connection timeout in seconds
const MongodbTimeout = 60
