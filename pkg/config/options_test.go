package config

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithRemoteLoader(t *testing.T) {
	t.Parallel()

	o := &options{}
	require.Nil(t, o.remoteLoader)

	loader := func(_ *RemoteSourceConfig) (io.Reader, error) {
		return strings.NewReader(`{}`), nil
	}

	WithRemoteLoader(loader)(o)
	require.NotNil(t, o.remoteLoader)
}

// TestLoad_withRemoteLoader verifies that a custom provider is delegated to the
// loader registered via WithRemoteLoader: the loader receives the resolved
// remote-source settings and the returned data overrides local file values.
func TestLoad_withRemoteLoader(t *testing.T) {
	t.Parallel()

	tmpConfigDir := t.TempDir()
	configContent := []byte(`
{
  "remoteConfigProvider": "customstore",
  "remoteConfigEndpoint": "store:1234",
  "remoteConfigPath": "/config/app",
  "string": "local_value",
  "int": 3
}`)
	require.NoError(t, os.WriteFile(filepath.Join(tmpConfigDir, "config.json"), configContent, 0o600))

	loader := func(rs *RemoteSourceConfig) (io.Reader, error) {
		require.Equal(t, "customstore", rs.Provider)
		require.Equal(t, "store:1234", rs.Endpoint)
		require.Equal(t, "/config/app", rs.Path)

		return strings.NewReader(`{"string":"remote_value"}`), nil
	}

	targetConfig := &testConfig{}

	err := Load("cmd", tmpConfigDir, "testloader", targetConfig, WithRemoteLoader(loader))
	require.NoError(t, err)
	require.Equal(t, "remote_value", targetConfig.String) // remote data overrides the local file
	require.Equal(t, 3, targetConfig.Int)                 // local values without remote override are preserved
}

// TestLoad_withRemoteLoaderError verifies that a failing remote loader surfaces
// as a wrapped ErrRemoteConfig.
func TestLoad_withRemoteLoaderError(t *testing.T) {
	t.Parallel()

	tmpConfigDir := t.TempDir()
	configContent := []byte(`{"remoteConfigProvider": "customstore"}`)
	require.NoError(t, os.WriteFile(filepath.Join(tmpConfigDir, "config.json"), configContent, 0o600))

	err := Load("cmd", tmpConfigDir, "testloadererr", &testConfig{}, WithRemoteLoader(func(_ *RemoteSourceConfig) (io.Reader, error) {
		return nil, errors.New("remote backend unavailable")
	}))
	require.Error(t, err)
	require.ErrorIs(t, err, ErrRemoteConfig)
}

// TestLoad_withRemoteLoaderNilReader verifies that a loader returning a nil
// reader with a nil error is treated as "no remote data to merge": Load
// succeeds with the local values.
func TestLoad_withRemoteLoaderNilReader(t *testing.T) {
	t.Parallel()

	tmpConfigDir := t.TempDir()
	configContent := []byte(`{"remoteConfigProvider": "customstore", "string": "local_value"}`)
	require.NoError(t, os.WriteFile(filepath.Join(tmpConfigDir, "config.json"), configContent, 0o600))

	targetConfig := &testConfig{}

	err := Load("cmd", tmpConfigDir, "testloadernilrd", targetConfig, WithRemoteLoader(func(_ *RemoteSourceConfig) (io.Reader, error) {
		return nil, nil //nolint:nilnil // the (nil, nil) "no remote data" contract is what this test exercises
	}))
	require.NoError(t, err)
	require.Equal(t, "local_value", targetConfig.String)
}

// closeRecordingReader wraps a reader and records whether Close was called.
type closeRecordingReader struct {
	io.Reader

	closed bool
}

func (c *closeRecordingReader) Close() error {
	c.closed = true
	return nil
}

// TestLoad_withRemoteLoaderClosesReader verifies that a loader-returned reader
// implementing io.Closer is closed after reading.
func TestLoad_withRemoteLoaderClosesReader(t *testing.T) {
	t.Parallel()

	tmpConfigDir := t.TempDir()
	configContent := []byte(`{"remoteConfigProvider": "customstore", "string": "local_value"}`)
	require.NoError(t, os.WriteFile(filepath.Join(tmpConfigDir, "config.json"), configContent, 0o600))

	rc := &closeRecordingReader{Reader: strings.NewReader(`{"string":"remote_value"}`)}
	targetConfig := &testConfig{}

	err := Load("cmd", tmpConfigDir, "testloaderclose", targetConfig, WithRemoteLoader(func(_ *RemoteSourceConfig) (io.Reader, error) {
		return rc, nil
	}))
	require.NoError(t, err)
	require.True(t, rc.closed)
	require.Equal(t, "remote_value", targetConfig.String)
}
