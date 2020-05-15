package main

import (
	"encoding/base64"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/spf13/viper"
)

// getTestCfgParams returns a valid reference configuration as the one parsed by Viper
func getTestCfgParams() *params {
	return &params{
		log: &LogData{
			Level:   "INFO",
			Network: "",
			Address: "",
		},
		quantity: 10,
	}
}

func TestCheckParams(t *testing.T) {
	err := checkParams(getTestCfgParams())
	if err != nil {
		t.Errorf("No errors are expected: %v", err)
	}
}

func TestCheckConfigParametersErrors(t *testing.T) {

	var testCases = []struct {
		fcfg  func(cfg *params) *params
		field string
	}{
		{func(cfg *params) *params { cfg.log.Level = ""; return cfg }, "log.Level"},
		{func(cfg *params) *params { cfg.log.Level = "INVALID"; return cfg }, "log.Level"},
		{func(cfg *params) *params { cfg.quantity = 0; return cfg }, "quantity"},
	}

	for _, tt := range testCases {
		cfg := getTestCfgParams()
		err := checkParams(tt.fcfg(cfg))
		if err == nil {
			t.Errorf("An error was expected because the %s field is invalid", tt.field)
		}
	}
}

func TestGetConfigParams(t *testing.T) {
	prm, err := getConfigParams()
	if err != nil {
		t.Errorf("An error was not expected: %v", err)
	}
	if prm.log.Level != "DEBUG" {
		t.Errorf("Found different logLevel than expected, found %s", prm.log.Level)
	}
}

func TestGetLocalConfigParams(t *testing.T) {

	// test environment variables
	defer unsetRemoteConfigEnv(t)
	setRemoteConfigEnv(t, []string{"consul", "127.0.0.1:98765", "/config/~#PROJECT#~", "", ""})

	prm, rprm, err := getLocalConfigParams()
	if err != nil {
		t.Errorf("An error was not expected: %v", err)
	}

	if prm.log.Level != "DEBUG" {
		t.Errorf("Found different logLevel than expected, found %s", prm.log.Level)
	}
	if rprm.remoteConfigProvider != "consul" {
		t.Errorf("Found different remoteConfigProvider than expected, found %s", rprm.remoteConfigProvider)
	}
	if rprm.remoteConfigEndpoint != "127.0.0.1:98765" {
		t.Errorf("Found different remoteConfigEndpoint than expected, found %s", rprm.remoteConfigEndpoint)
	}
	if rprm.remoteConfigPath != "/config/~#PROJECT#~" {
		t.Errorf("Found different remoteConfigPath than expected, found %s", rprm.remoteConfigPath)
	}
	if rprm.remoteConfigSecretKeyring != "" {
		t.Errorf("Found different remoteConfigSecretKeyring than expected, found %s", rprm.remoteConfigSecretKeyring)
	}

	_, err = getRemoteConfigParams(prm, rprm)
	if err == nil {
		t.Errorf("A remote configuration error was expected")
	}

	rprm.remoteConfigSecretKeyring = "/etc/~#PROJECT#~/cfgkey.gpg"
	_, err = getRemoteConfigParams(prm, rprm)
	if err == nil {
		t.Errorf("A remote configuration error was expected")
	}
}

func TestGetConfigParamsRemoteEnv(t *testing.T) {

	data, err := ioutil.ReadFile("../resources/test/etc/~#PROJECT#~/env.config.json")
	if err != nil {
		t.Errorf("Unable to read the env.config.json file: %v", err)
	}
	envdata := base64.StdEncoding.EncodeToString(data)

	// test environment variables
	defer unsetRemoteConfigEnv(t)
	setRemoteConfigEnv(t, []string{"envvar", "", "", "", envdata})

	viper.Reset()
	prm, err := getConfigParams()
	if err != nil {
		t.Errorf("An error was not expected: %v", err)
	}
	if prm.quantity != 13 {
		t.Errorf("Expected quantity=13, found %d", prm.quantity)
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
	defer unsetRemoteConfigEnv(t)
	setRemoteConfigEnv(t, []string{"consul", "127.0.0.1:8500", "/config/~#PROJECT#~", "", ""})

	// load a specific config file just for testing
	oldCfg := ConfigPath
	viper.Reset()
	for k := range ConfigPath {
		ConfigPath[k] = "wrong/path/"
	}
	defer func() { ConfigPath = oldCfg }()

	prm, err := getConfigParams()
	if err != nil {
		t.Errorf("An error was not expected: %v", err)
	}
	if prm.log.Level != "debug" {
		t.Errorf("Found different logLevel than expected, found %s", prm.log.Level)
	}
}

func TestCliWrongConfigError(t *testing.T) {

	// test environment variables
	defer unsetRemoteConfigEnv(t)
	setRemoteConfigEnv(t, []string{"consul", "127.0.0.1:999999", "/config/wrong", "", ""})

	// load a specific config file just for testing
	oldCfg := ConfigPath
	viper.Reset()
	for k := range ConfigPath {
		ConfigPath[k] = "wrong/path/"
	}
	defer func() { ConfigPath = oldCfg }()

	cmd, err := cli()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if cmdtype := reflect.TypeOf(cmd).String(); cmdtype != "*cobra.Command" {
		t.Errorf("The expected type is '*cobra.Command', found: '%s'", cmdtype)
		return
	}

	old := os.Stderr // keep backup of the real stdout
	defer func() { os.Stderr = old }()
	os.Stderr = nil

	// execute the main function
	if err := cmd.Execute(); err == nil {
		t.Errorf("An error was expected")
	}
}

func unsetRemoteConfigEnv(t *testing.T) {
	setRemoteConfigEnv(t, []string{"", "", "", "", ""})
}

func setRemoteConfigEnv(t *testing.T, val []string) {
	envVar := []string{
		"~#UPROJECT#~_REMOTECONFIGPROVIDER",
		"~#UPROJECT#~_REMOTECONFIGENDPOINT",
		"~#UPROJECT#~_REMOTECONFIGPATH",
		"~#UPROJECT#~_REMOTECONFIGSECRETKEYRING",
		"~#UPROJECT#~_REMOTECONFIGDATA",
	}
	for i, ev := range envVar {
		err := os.Setenv(ev, val[i])
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}
