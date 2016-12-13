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

// Log (syslog)

// LogLevel defines the default log level: EMERGENCY, ALERT, CRITICAL, ERROR, WARNING, NOTICE, INFO, DEBUG
const LogLevel = "INFO"

// LogNetwork is the network type used by Syslog (i.e. udp or tcp). Leave emty to disable.
const LogNetwork = ""

// LogAddress is the network address of the Syslog daemon (ip:port) or just (:port). Leave emty to disable.
const LogAddress = ""

// Quantity is the default number of results to return
const Quantity = 1
