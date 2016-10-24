package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/spf13/viper"
)

func TestCheckParams(t *testing.T) {
	err := checkParams(&params{
		quantity: 10,
		logLevel: "info",
	})
	if err != nil {
		t.Error(fmt.Errorf("No errors are expected"))
	}
}

func TestCheckParamsErrorsServer(t *testing.T) {
	err := checkParams(&params{quantity: 0})
	if err == nil {
		t.Error(fmt.Errorf("An error was expected because the quantity is <= 0"))
	}
}

func TestCheckParamsErrorsLogLevelEmpty(t *testing.T) {
	err := checkParams(&params{
		quantity: 10,
		logLevel: "",
	})
	if err == nil {
		t.Error(fmt.Errorf("An error was expected because the logLevel is empty"))
	}
}

func TestCheckParamsErrorsLogLevelInvalid(t *testing.T) {
	err := checkParams(&params{
		quantity: 10,
		logLevel: "INVALID",
	})
	if err == nil {
		t.Error(fmt.Errorf("An error was expected because the logLevel is not valid"))
	}
}

func TestGetConfigParams(t *testing.T) {
	prm, err := getConfigParams()
	if err != nil {
		t.Error(fmt.Errorf("An error was not expected: %v", err))
	}
	if prm.logLevel != "debug" {
		t.Error(fmt.Errorf("Found different logLevel than expected, found %s", prm.logLevel))
	}
}

func TestGetLocalConfigParams(t *testing.T) {

	// test environment variables
	defer unsetRemoteConfigEnv()
	os.Setenv("~#UPROJECT#~_REMOTECONFIGPROVIDER", "consul")
	os.Setenv("~#UPROJECT#~_REMOTECONFIGENDPOINT", "127.0.0.1:98765")
	os.Setenv("~#UPROJECT#~_REMOTECONFIGPATH", "/config/~#PROJECT#~")
	os.Setenv("~#UPROJECT#~_REMOTECONFIGSECRETKEYRING", "")

	prm, rprm := getLocalConfigParams()

	if prm.logLevel != "debug" {
		t.Error(fmt.Errorf("Found different logLevel than expected, found %s", prm.logLevel))
	}
	if rprm.remoteConfigProvider != "consul" {
		t.Error(fmt.Errorf("Found different remoteConfigProvider than expected, found %s", rprm.remoteConfigProvider))
	}
	if rprm.remoteConfigEndpoint != "127.0.0.1:98765" {
		t.Error(fmt.Errorf("Found different remoteConfigEndpoint than expected, found %s", rprm.remoteConfigEndpoint))
	}
	if rprm.remoteConfigPath != "/config/~#PROJECT#~" {
		t.Error(fmt.Errorf("Found different remoteConfigPath than expected, found %s", rprm.remoteConfigPath))
	}
	if rprm.remoteConfigSecretKeyring != "" {
		t.Error(fmt.Errorf("Found different remoteConfigSecretKeyring than expected, found %s", rprm.remoteConfigSecretKeyring))
	}

	_, err := getRemoteConfigParams(prm, rprm)
	if err == nil {
		t.Error(fmt.Errorf("A remote configuration error was expected"))
	}

	rprm.remoteConfigSecretKeyring = "/etc/~#PROJECT#~/cfgkey.gpg"
	_, err = getRemoteConfigParams(prm, rprm)
	if err == nil {
		t.Error(fmt.Errorf("A remote configuration error was expected"))
	}
}

// Test real Consul provider
// To activate this define the environmental variable ~#UPROJECT#~_LIVECONSUL
func TestGetConfigParamsRemote(t *testing.T) {

	enable := os.Getenv("~#UPROJECT#~_LIVECONSUL")
	if enable == "" {
		return
	}

	// test environment variables
	defer unsetRemoteConfigEnv()
	os.Setenv("~#UPROJECT#~_REMOTECONFIGPROVIDER", "consul")
	os.Setenv("~#UPROJECT#~_REMOTECONFIGENDPOINT", "127.0.0.1:8500")
	os.Setenv("~#UPROJECT#~_REMOTECONFIGPATH", "/config/~#PROJECT#~")
	os.Setenv("~#UPROJECT#~_REMOTECONFIGSECRETKEYRING", "")

	// load a specific config file just for testing
	oldCfg := ConfigPath
	viper.Reset()
	for k := range ConfigPath {
		ConfigPath[k] = "wrong/path/"
	}
	defer func() { ConfigPath = oldCfg }()

	prm, err := getConfigParams()
	if err != nil {
		t.Error(fmt.Errorf("An error was not expected: %v", err))
	}
	if prm.logLevel != "debug" {
		t.Error(fmt.Errorf("Found different logLevel than expected, found %s", prm.logLevel))
	}
}

// unsetRemoteConfigEnv clear the environmental variables used to set the remote configuration
func unsetRemoteConfigEnv() {
	os.Setenv("~#UPROJECT#~_REMOTECONFIGPROVIDER", "")
	os.Setenv("~#UPROJECT#~_REMOTECONFIGENDPOINT", "")
	os.Setenv("~#UPROJECT#~_REMOTECONFIGPATH", "")
	os.Setenv("~#UPROJECT#~_REMOTECONFIGSECRETKEYRING", "")
}
