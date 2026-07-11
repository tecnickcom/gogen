package config

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// fakeViperRemoteFactory implements the interface behind viper.RemoteConfig so
// ViperRemoteLoader can be tested without importing
// github.com/spf13/viper/remote (which would drag its backend client
// dependencies into this module).
type fakeViperRemoteFactory struct {
	getFn func(rp viper.RemoteProvider) (io.Reader, error)
}

func (f *fakeViperRemoteFactory) Get(rp viper.RemoteProvider) (io.Reader, error) {
	return f.getFn(rp)
}

func (f *fakeViperRemoteFactory) Watch(rp viper.RemoteProvider) (io.Reader, error) {
	return f.getFn(rp)
}

func (f *fakeViperRemoteFactory) WatchChannel(_ viper.RemoteProvider) (<-chan *viper.RemoteResponse, chan bool) {
	return nil, nil
}

// TestViperRemoteLoader verifies that the legacy Viper remote path is used
// when viper.RemoteConfig is registered: the adapter forwards the resolved
// remote-source settings and the fetched data overrides local file values.
//
//nolint:paralleltest // mutates the global viper.RemoteConfig
func TestViperRemoteLoader(t *testing.T) {
	oldFactory := viper.RemoteConfig

	t.Cleanup(func() { viper.RemoteConfig = oldFactory })

	viper.RemoteConfig = &fakeViperRemoteFactory{
		getFn: func(rp viper.RemoteProvider) (io.Reader, error) {
			require.Equal(t, "consul", rp.Provider())
			require.Equal(t, "store:1234", rp.Endpoint())
			require.Equal(t, "/config/app", rp.Path())
			require.Equal(t, "/etc/app/configkey.gpg", rp.SecretKeyring())

			return strings.NewReader(`{"string":"viper_remote_value"}`), nil
		},
	}

	tmpConfigDir := t.TempDir()
	configContent := []byte(`
{
  "remoteConfigProvider": "consul",
  "remoteConfigEndpoint": "store:1234",
  "remoteConfigPath": "/config/app",
  "remoteConfigSecretKeyring": "/etc/app/configkey.gpg",
  "string": "local_value"
}`)
	require.NoError(t, os.WriteFile(filepath.Join(tmpConfigDir, "config.json"), configContent, 0o600))

	targetConfig := &testConfig{}

	err := Load("cmd", tmpConfigDir, "testviperloader", targetConfig, WithRemoteLoader(ViperRemoteLoader))
	require.NoError(t, err)
	require.Equal(t, "viper_remote_value", targetConfig.String)
}

// TestViperRemoteLoader_notRegistered verifies that using ViperRemoteLoader
// without the github.com/spf13/viper/remote blank import fails with
// ErrViperRemoteNotRegistered.
//
//nolint:paralleltest // mutates the global viper.RemoteConfig
func TestViperRemoteLoader_notRegistered(t *testing.T) {
	oldFactory := viper.RemoteConfig

	t.Cleanup(func() { viper.RemoteConfig = oldFactory })

	viper.RemoteConfig = nil

	tmpConfigDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpConfigDir, "config.json"), []byte(`{"remoteConfigProvider": "consul"}`), 0o600))

	err := Load("cmd", tmpConfigDir, "testviperloadernil", &testConfig{}, WithRemoteLoader(ViperRemoteLoader))
	require.Error(t, err)
	require.ErrorIs(t, err, ErrViperRemoteNotRegistered)
	require.ErrorIs(t, err, ErrRemoteConfig)
}

// TestViperRemoteLoader_validation verifies the fail-fast checks: unsupported
// provider names are rejected (the viper remote package would otherwise
// silently route them to the consul backend) and empty endpoint/path are
// rejected (the backend client would otherwise fall back to its default
// address).
//
//nolint:paralleltest // mutates the global viper.RemoteConfig
func TestViperRemoteLoader_validation(t *testing.T) {
	oldFactory := viper.RemoteConfig

	t.Cleanup(func() { viper.RemoteConfig = oldFactory })

	viper.RemoteConfig = &fakeViperRemoteFactory{
		getFn: func(_ viper.RemoteProvider) (io.Reader, error) {
			return nil, errors.New("the backend must not be reached on validation failure")
		},
	}

	_, err := ViperRemoteLoader(&RemoteSourceConfig{Provider: "vault", Endpoint: "remote:1234", Path: "/config/path"})
	require.ErrorIs(t, err, ErrInvalidRemoteConfig)

	_, err = ViperRemoteLoader(&RemoteSourceConfig{Provider: "consul", Path: "/config/path"})
	require.ErrorIs(t, err, ErrMissingRemoteVar)

	_, err = ViperRemoteLoader(&RemoteSourceConfig{Provider: "consul", Endpoint: "remote:1234"})
	require.ErrorIs(t, err, ErrMissingRemoteVar)
}
