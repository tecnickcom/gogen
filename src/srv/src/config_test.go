package main

import (
	"encoding/base64"
	"io/ioutil"
	"net/url"
	"os"
	"reflect"
	"testing"

	"github.com/spf13/viper"
)

// getTestCfgParams returns a valid reference configuration as the one parsed by Viper
func getTestCfgParams() *params {
	return &params{
		serverAddress: ":8123",
		tls: &TLSData{
			Enabled: true,
			CertPem: []byte("-----BEGIN CERTIFICATE-----\nMIIFPzCCAyegAwIBAgIUODMFtHLbGfFtm9dstcbp9ZvFrhkwDQYJKoZIhvcNAQEL\nBQAwLzELMAkGA1UEBhMCVUsxEzARBgNVBAgMClNvbWUtU3RhdGUxCzAJBgNVBAoM\nAlRDMB4XDTIwMDExMjEyNTY1OFoXDTMwMDEwOTEyNTY1OFowLzELMAkGA1UEBhMC\nVUsxEzARBgNVBAgMClNvbWUtU3RhdGUxCzAJBgNVBAoMAlRDMIICIjANBgkqhkiG\n9w0BAQEFAAOCAg8AMIICCgKCAgEA1UhdFDYrGdHm2ymfyWTDmeVT9qdeIgaDZ5+m\njCE6PCDGL3eDuzo+lkSBL/MiFG3BjxoItLp4HvFs5gesuxpIjbNir2LSEKi9QO3+\n54VrHICRgtmgc+PkanRn1PwrkTR53Vg1QBy7UGJleAu6EgBvv2Tc2yp6PPPLIW6w\n9GVpTY6FdSXrPQsRz2coVaXQ6zdcsueTaeqCxgyhy8ybaRevPd4Whua7XXpXIGKs\nQg9/wRVvjdTzttJzZnv9e8+pQU/AHg8H04OS9oBGKnHqOD4qwCpxzQ3+hA22FKKb\nR4sASN40ONLf67Gbg05nSM5nnZtIH578vNVC6yYS/elm9lt3cmjyxdrsOxjFNkBb\nJ1c/LIk9SocGkI5m4bWElr2jvEfDLIkDz5XFzN24vmnoS7widsFIvJ7kr4mmggMh\nPw++eqBvQcGEqSJt7PIeqEGxSC0fO2V4ZnKOSNQQk8++Dd4i73hdhAnwfjTy3cEM\n6YGjdKzO7fwRhSxXy6YP9MyDKQHe6NoWq0H7akPo5iD3CvJ976Yi7EaSmcJO7msv\nk34RqO9xyFnkj2UAWjgitR8wDZVaKQb6MFpZUBwmcMxvCOvvxrG5/2bVZMZHLVCB\n9u9x8ge4hlhTeXWJIXTzySgWDEAZCxhGXubo5RE6MvMB8d1FoT25tZSaxBvVq9kT\n6xqNzQ8CAwEAAaNTMFEwHQYDVR0OBBYEFFUA1Xnu9+mI4Ji9BNgoTWHrWRlnMB8G\nA1UdIwQYMBaAFFUA1Xnu9+mI4Ji9BNgoTWHrWRlnMA8GA1UdEwEB/wQFMAMBAf8w\nDQYJKoZIhvcNAQELBQADggIBABlcDM2jK1e24REBaaEZLszvmR8s83HVdGMiwcRQ\nq/I4hLBQLj8gmDDSB+cDf2qH3/bjSycdgtdVPLih2Rl5/JsLXvCLRGHAbu/VizCF\nNJ0Vuq0xMEQcuxn4WU/r74af8J+gbxW729LquHfreUdXCvenEIdnCD3aPw7Xh0R0\nbX2p9exbIQR+KhXbvReHMbnZ0AsW0JmLDqNgwAauDwVAC68WkVhUYJxYKAQHByg/\nU1auMNvnwWL0Soiiq2skTBv44+6OATVRi7VrN/ZbZl+SvnLAsOTFan+cWyL8i3iC\nsUz0mEqkdQtMPY9IchHaCoC9TDGwRLzK0q0mkeEb2dWgfsUrqzrTakk2xcTX/iZn\nErU1Fj/zhunyi0E7zJOkSMKLwTy5tZ4/UPld/Bv/QlZRsRnEKC+KwW5qNyTcIHR7\nJShxtSqNHV4K0uVcMwX23S8dKCJU8x/YKR7c+mlrkUH2OAcVN/kfxzXA+6qWsHjr\nEE53o684+Uex9gkfDerXADbEvdAcsfNOgmtc5YyWnquJWcWGM5zIgjqO7wvrj62D\nMSsLHQ5WyLjCm24LQ3eMu8jaLiiZ267vyRaqGdzGQikgKY1Bx4TrajazPNCn8Wq2\nyyzLUnP+b8bft3tPf1O5JQgX634Xc/rPWHEbs9bei9esYIXza0upORTHiA5Dz6F8\ngTPW\n-----END CERTIFICATE-----"),
			KeyPem:  []byte("-----BEGIN PRIVATE KEY-----\n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\nFB7zOBXRuj9X4Vecfkh9hfFmuvdYTQw=\n-----END PRIVATE KEY-----"),
		},
		log: &LogData{
			Level:   "INFO",
			Network: "",
			Address: "",
		},
		stats: &StatsData{
			Prefix:      "~#PROJECT#~-test",
			Network:     "udp",
			Address:     ":8125",
			FlushPeriod: 100,
		},
		user: map[string]string{
			"test": "$2a$04$GfYChjSytr0zgLYbSJoyK.XZGbiNm4F5VY08WL0bHBAKgnq3AkcZu",
		},
		jwt: &JwtData{
			Enabled:   true,
			Key:       []byte("4VjfMNj3TKC2xrhL5vKzjZDAkAaGn8GwHdT6zyF8tGeStvv35jS3wHFyrHYF5VejDvs5Jsa5E52GrfwbHW9fwzbQYm7kTJaaR8SfwGVrxvRsTM3L6rMEYPUm3ammHzeSYb8Hn5wbs2tMxtrx4dufUhYae9eYm9X77QH5WA8EcSThzZVvbMba3MrDmPrZxGBat9WFVjvLPhrpyFkeSDNJEVXZN6AynF7DPWPBJBasZvxuuAGQzgjeUUmRJDn3xP9a"),
			Exp:       5,
			RenewTime: 299,
		},
		proxyAddress: "https://localhost:8117/test",
		proxyURL: &url.URL{
			Scheme: "http",
			Host:   "localhost:8117",
			Path:   "/",
		},
		mysql: &MysqlData{
			DSN: "username:password@protocol(address)/dbname?param=value",
		},
		mongodb: &MongodbData{
			Address:  "1.2.3.4:5432",
			Database: "testdb",
			User:     "dbuser",
			Password: "dbpassword",
			Timeout:  60,
		},
		elasticsearch: &ElasticsearchData{
			URL:      "http://127.0.0.1:9200",
			Index:    "test",
			Username: "alpha",
			Password: "beta",
		},
	}
}

func TestCheckParams(t *testing.T) {
	err := checkParams(getTestCfgParams())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestCheckConfigParametersErrors(t *testing.T) {

	var testCases = []struct {
		fcfg  func(cfg *params) *params
		field string
	}{
		{func(cfg *params) *params { cfg.log.Level = ""; return cfg }, "log.Level"},
		{func(cfg *params) *params { cfg.log.Level = "INVALID"; return cfg }, "log.Level"},
		{func(cfg *params) *params { cfg.serverAddress = ""; return cfg }, "serverAddress"},
		{func(cfg *params) *params { cfg.tls.Enabled = true; cfg.tls.CertPem = []byte(""); return cfg }, "tls.certPem"},
		{func(cfg *params) *params { cfg.tls.Enabled = true; cfg.tls.KeyPem = []byte(""); return cfg }, "tls.keyPem"},
		{func(cfg *params) *params { cfg.stats.Prefix = ""; return cfg }, "stats.Prefix"},
		{func(cfg *params) *params { cfg.stats.Network = ""; return cfg }, "stats.Network"},
		{func(cfg *params) *params { cfg.stats.FlushPeriod = -1; return cfg }, "stats.FlushPeriod"},
		{func(cfg *params) *params { cfg.user = map[string]string{}; return cfg }, "user"},
		{func(cfg *params) *params { cfg.jwt.Enabled = true; cfg.jwt.Key = []byte(""); return cfg }, "jwt.key"},
		{func(cfg *params) *params { cfg.jwt.Enabled = true; cfg.jwt.Exp = -1; return cfg }, "jwt.exp"},
		{func(cfg *params) *params { cfg.jwt.Enabled = true; cfg.jwt.RenewTime = -1; return cfg }, "jwt.renewTime"},
		{func(cfg *params) *params { cfg.proxyAddress = ""; return cfg }, "proxyAddress"},
		{func(cfg *params) *params { cfg.proxyAddress = "+*&^https://error\"!£"; return cfg }, "proxyAddress"},
		{func(cfg *params) *params {
			cfg.mongodb.Address = "127.0.0.1:98765"
			cfg.mongodb.Database = ""
			return cfg
		}, "mongodb.Database"},
		{func(cfg *params) *params {
			cfg.mongodb.Address = "127.0.0.1:98765"
			cfg.mongodb.Timeout = 0
			return cfg
		}, "mongodb.Timeout"},
		{func(cfg *params) *params {
			cfg.elasticsearch.URL = "127.0.0.1:56789"
			cfg.elasticsearch.Index = ""
			return cfg
		}, "elasticsearch.Index"},
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
	if prm.serverAddress != ":8017" {
		t.Errorf("Found different server address than expected, found %s", prm.serverAddress)
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

	if prm.serverAddress != ":8017" {
		t.Errorf("Found different server address than expected, found %s", prm.serverAddress)
	}
	if prm.proxyAddress != "http://localhost:8117" {
		t.Errorf("Found different proxy address than expected, found %s", prm.proxyAddress)
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
	if prm.jwt.RenewTime != 234 {
		t.Errorf("Expected jwt.renewTime 234, found %d", prm.jwt.RenewTime)
	}
	if prm.stats.FlushPeriod != 117 {
		t.Errorf("Expected stats.FlushPeriod 117, found %d", prm.stats.FlushPeriod)
	}
}

// Test real Consul provider
// To activate this define the environmental variable ~#UPROJECT#~_LIVECONSUL
func TestGetConfigParamsRemoteConsul(t *testing.T) {

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
	if prm.serverAddress != ":8017" {
		t.Errorf("Found different serverAddress than expected, found %s", prm.serverAddress)
	}
	if prm.log.Level != "DEBUG" {
		t.Errorf("Found different log.Level than expected, found %s", prm.log.Level)
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
